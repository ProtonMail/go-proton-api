package proton

import (
	"context"

	"github.com/go-resty/resty/v2"
)

func (c *Client) GetLink(ctx context.Context, shareID, linkID string) (Link, error) {
	var res struct {
		Link Link
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetResult(&res).Get("/drive/shares/" + shareID + "/links/" + linkID)
	}); err != nil {
		return Link{}, err
	}

	return res.Link, nil
}
