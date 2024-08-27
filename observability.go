package proton

import (
	"context"

	"github.com/go-resty/resty/v2"
)

func (c *Client) SendObservabilityBatch(ctx context.Context, req ObservabilityBatch) error {
	return c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetHeader("Priority", "u=6").SetBody(req).Post("/data/v1/metrics")
	})
}
