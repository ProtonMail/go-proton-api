package proton

import (
	"context"
	"fmt"

	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/go-resty/resty/v2"
)

func (c *Client) CreateDraft(ctx context.Context, addrKR *crypto.KeyRing, req CreateDraftReq) (Message, error) {
	var res struct {
		Message Message
	}

	enc, err := addrKR.Encrypt(crypto.NewPlainMessageFromString(req.Message.Body), nil)
	if err != nil {
		return Message{}, fmt.Errorf("failed to encrypt draft: %w", err)
	}

	arm, err := enc.GetArmored()
	if err != nil {
		return Message{}, fmt.Errorf("failed to armor draft: %w", err)
	}

	type encCreateDraftReq struct {
		CreateDraftReq

		Body string
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetBody(encCreateDraftReq{
			CreateDraftReq: req,
			Body:           arm,
		}).SetResult(&res).Post("/mail/v4/messages")
	}); err != nil {
		return Message{}, err
	}

	return res.Message, nil
}

func (c *Client) UpdateDraft(ctx context.Context, draftID string, addrKR *crypto.KeyRing, req UpdateDraftReq) (Message, error) {
	var res struct {
		Message Message
	}

	var encBody string

	if req.Message.Body != "" {
		enc, err := addrKR.Encrypt(crypto.NewPlainMessageFromString(req.Message.Body), nil)
		if err != nil {
			return Message{}, fmt.Errorf("failed to encrypt draft: %w", err)
		}

		arm, err := enc.GetArmored()
		if err != nil {
			return Message{}, fmt.Errorf("failed to armor draft: %w", err)
		}

		encBody = arm
	}

	type encUpdateDraftReq struct {
		UpdateDraftReq

		Body string
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetBody(encUpdateDraftReq{
			UpdateDraftReq: req,
			Body:           encBody,
		}).SetResult(&res).Put("/mail/v4/messages/" + draftID)
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
