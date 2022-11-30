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

const maxImportSize = 10

func (c *Client) ImportMessages(ctx context.Context, addrKR *crypto.KeyRing, workers, buffer int, req ...ImportReq) stream.Stream[ImportRes] {
	return stream.Flatten(parallel.MapStream(
		ctx,
		stream.FromIterator(iterator.Chunk(iterator.Slice(req), maxImportSize)),
		workers,
		buffer,
		func(ctx context.Context, req []ImportReq) (stream.Stream[ImportRes], error) {
			res, err := c.importMessages(ctx, addrKR, req)
			if err != nil {
				return nil, fmt.Errorf("failed to import messages: %w", err)
			}

			for _, res := range res {
				if res.Code != SuccessCode {
					return nil, fmt.Errorf("failed to import message: %w", res.Error)
				}
			}

			return stream.FromIterator(iterator.Slice(res)), nil
		},
	))
}

func (c *Client) importMessages(ctx context.Context, addrKR *crypto.KeyRing, req []ImportReq) ([]ImportRes, error) {
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
		fields, err := buildImportReqFields(addrKR, named)
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
