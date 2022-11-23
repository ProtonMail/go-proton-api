package proton

import (
	"context"
	"io"

	"github.com/go-resty/resty/v2"
)

func (c *Client) GetBlock(ctx context.Context, url string) (io.ReadCloser, error) {
	res, err := c.doRes(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetDoNotParseResponse(true).Get(url)
	})
	if err != nil {
		return nil, err
	}

	return res.RawBody(), nil
}
