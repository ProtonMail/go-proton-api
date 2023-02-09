package proton

import (
	"context"
	"fmt"
	"strconv"

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

	// maxImportSize is the maximum total request size permitted for a single import request.
	maxImportSize = 30 * 1024 * 1024
)

func (c *Client) ImportMessages(ctx context.Context, addrKR *crypto.KeyRing, workers, buffer int, req ...ImportReq) (stream.Stream[ImportRes], error) {
	for idx := range req {
		enc, err := EncryptRFC822(addrKR, req[idx].Message)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt message %v: %w", idx, err)
		}

		req[idx].Message = enc
	}

	return stream.Flatten(parallel.MapStream(
		ctx,
		stream.FromIterator(iterator.Slice(chunkSized(req, maxImportCount, maxImportSize, func(req ImportReq) int {
			return len(req.Message)
		}))),
		workers,
		buffer,
		func(ctx context.Context, req []ImportReq) (stream.Stream[ImportRes], error) {
			res, err := c.importMessages(ctx, req)
			if err != nil {
				return nil, fmt.Errorf("failed to import messages: %w", err)
			}

			for _, res := range res {
				if res.Code != SuccessCode {
					return nil, fmt.Errorf("failed to import message: %w", res.APIError)
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

// chunkSized splits a slice into chunks of maximum size and length.
func chunkSized[T any](vals []T, maxLen, maxSize int, getSize func(T) int) [][]T {
	var chunks [][]T

	for len(vals) > 0 {
		var (
			curChunk []T
			curSize  int
		)

		for len(vals) > 0 && len(curChunk) < maxLen && curSize < maxSize {
			val, size := vals[0], getSize(vals[0])

			if curSize+size <= maxSize {
				curChunk = append(curChunk, val)
				curSize += size
				vals = vals[1:]
			} else if len(curChunk) == 0 {
				curChunk = append(curChunk, val)
				curSize += size
				vals = vals[1:]
			} else {
				break
			}
		}

		chunks = append(chunks, curChunk)
	}

	return chunks
}
