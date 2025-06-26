package proton

import (
	"context"

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
)

func (c *Client) GetFeatures(ctx context.Context, stickyKey uuid.UUID) (FeatureFlagResult, error) {
	res := FeatureFlagResult{}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetResult(&res).Get(getFeatureFlagEndpoint(stickyKey))
	}); err != nil {
		return FeatureFlagResult{}, err
	}

	return res, nil
}
