package proton

import (
	"context"

	"github.com/go-resty/resty/v2"
	"golang.org/x/exp/slices"
)

func (c *Client) GetAddresses(ctx context.Context) ([]Address, error) {
	var res struct {
		Addresses []Address
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetResult(&res).Get("/core/v4/addresses")
	}); err != nil {
		return nil, err
	}

	slices.SortFunc(res.Addresses, func(a, b Address) bool {
		return a.Order < b.Order
	})

	return res.Addresses, nil
}

func (c *Client) GetAddress(ctx context.Context, addressID string) (Address, error) {
	var res struct {
		Address Address
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetResult(&res).Get("/core/v4/addresses/" + addressID)
	}); err != nil {
		return Address{}, err
	}

	return res.Address, nil
}

func (c *Client) OrderAddresses(ctx context.Context, req OrderAddressesReq) error {
	return c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetBody(req).Put("/core/v4/addresses/order")
	})
}
