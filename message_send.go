package proton

import (
	"context"

	"github.com/go-resty/resty/v2"
)

func (c *Client) CreateDraft(ctx context.Context, req CreateDraftReq) (Message, error) {
	var res struct {
		Message Message
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetBody(req).SetResult(&res).Post("/mail/v4/messages")
	}); err != nil {
		return Message{}, err
	}

	return res.Message, nil
}

func (c *Client) SendDraft(ctx context.Context, draftID string, req SendDraftReq) (Message, error) {
	var res struct {
		Sent Message
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetBody(req).SetResult(&res).Post("/mail/v4/messages/" + draftID)
	}); err != nil {
		return Message{}, err
	}

	return res.Sent, nil
}
