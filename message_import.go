package proton

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/ProtonMail/gluon/async"
	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/bradenaw/juniper/iterator"
	"github.com/bradenaw/juniper/parallel"
	"github.com/bradenaw/juniper/stream"
	"github.com/bradenaw/juniper/xslices"
	"github.com/go-resty/resty/v2"
)

const (
	// maxImportCount is the maximum number of messages that can be imported in a single request.
	maxImportCount = 10

	// MaxImportSize is the maximum total request size permitted for a single import request.
	MaxImportSize = 30 * 1024 * 1024
)

var ErrImportEncrypt = errors.New("failed to encrypt message")
var ErrImportSizeExceeded = errors.New("message exceeds maximum import size of 30MB")

type ImportResStream stream.Stream[ImportRes] // gomock does not support generics. In order to be able to mock ImportMessages, we introduce a typedef.

func (c *Client) ImportMessages(ctx context.Context, addrKR *crypto.KeyRing, workers, buffer int, req ...ImportReq) (ImportResStream, error) {
	// Encrypt each message.
	for idx := range req {

		// Encryption might mangle the original message bufer, so use a copy.
		msgCopy := make([]byte, len(req[idx].Message))
		copy(msgCopy, req[idx].Message)

		enc, err := EncryptRFC822(addrKR, msgCopy)
		if err != nil {
			return nil, fmt.Errorf("%w %v: %v", ErrImportEncrypt, idx, err)
		}

		req[idx].encryptedMessage = enc
	}

	// If any of the messages exceed the maximum import size, return an error.
	if xslices.Any(req, func(req ImportReq) bool { return len(req.encryptedMessage) > MaxImportSize }) {
		return nil, ErrImportSizeExceeded
	}

	return stream.Flatten(parallel.MapStream(
		ctx,
		stream.FromIterator(iterator.Slice(ChunkSized(req, maxImportCount, MaxImportSize, func(req ImportReq) int {
			return len(req.encryptedMessage)
		}))),
		workers,
		buffer,
		func(ctx context.Context, req []ImportReq) (stream.Stream[ImportRes], error) {
			defer async.HandlePanic(c.m.panicHandler)

			res, err := c.importMessages(ctx, req)
			if err != nil {
				return nil, fmt.Errorf("failed to import messages: %w", err)
			}

			for _, res := range res {
				if res.Code != SuccessCode {
					return nil, fmt.Errorf("failed to import message: %w", res)
				}
			}

			return stream.FromIterator(iterator.Slice(res)), nil
		},
	)), nil
}

func (c *Client) importMessages(ctx context.Context, req []ImportReq) ([]ImportRes, error) {
	names := iterator.Collect(iterator.Map(iterator.Counter(len(req)), func(i int) string {
		return strconv.Itoa(i)
	}))

	var named []namedImportReq

	for idx, name := range names {
		named = append(named, namedImportReq{
			ImportReq: req[idx],
			Name:      name,
		})
	}

	type namedImportRes struct {
		Name     string
		Response ImportRes
	}

	var res struct {
		Responses []namedImportRes
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		fields, err := buildImportReqFields(named)
		if err != nil {
			return nil, err
		}

		return r.SetMultipartFields(fields...).SetResult(&res).Post("/mail/v4/messages/import")
	}); err != nil {
		return nil, err
	}

	namedRes := make(map[string]ImportRes, len(res.Responses))

	for _, res := range res.Responses {
		namedRes[res.Name] = res.Response
	}

	return xslices.Map(names, func(name string) ImportRes {
		return namedRes[name]
	}), nil
}

// ChunkSized splits a slice into chunks of maximum size and length.
// It is assumed that the size of each element is less than the maximum size.
func ChunkSized[T any](vals []T, maxLen, maxSize int, getSize func(T) int) [][]T {
	var chunks [][]T

	for len(vals) > 0 {
		var (
			curChunk []T
			curSize  int
		)

		for len(vals) > 0 && len(curChunk) < maxLen && curSize < maxSize {
			val, size := vals[0], getSize(vals[0])

			if curSize+size > maxSize {
				break
			}

			curChunk = append(curChunk, val)
			curSize += size
			vals = vals[1:]
		}

		chunks = append(chunks, curChunk)
	}

	return chunks
}
