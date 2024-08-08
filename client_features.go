package proton

import (
	"context"

	"github.com/go-resty/resty/v2"
)

func (c *Client) GetFeatures(ctx context.Context) (FeatureFlagResult, error) {
	res := FeatureFlagResult{}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetResult(&res).Get("/feature/v2/frontend")
	}); err != nil {
		return FeatureFlagResult{}, err
	}

	return res, nil
}
