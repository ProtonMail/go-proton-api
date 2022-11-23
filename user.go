package proton

import (
	"context"

	"github.com/go-resty/resty/v2"
)

func (c *Client) GetUser(ctx context.Context) (User, error) {
	var res struct {
		User User
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetResult(&res).Get("/core/v4/users")
	}); err != nil {
		return User{}, err
	}

	return res.User, nil
}

func (c *Client) SendVerificationCode(ctx context.Context, req SendVerificationCodeReq) error {
	return c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetBody(req).Post("/core/v4/users/code")
	})
}

func (c *Client) CreateUser(ctx context.Context, req CreateUserReq) (User, error) {
	var res struct {
		User User
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetBody(req).SetResult(&res).Post("/core/v4/users")
	}); err != nil {
		return User{}, err
	}

	return res.User, nil
}

func (c *Client) GetUsernameAvailable(ctx context.Context, username string) error {
	return c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetQueryParam("Name", username).Get("/core/v4/users/available")
	})
}
