package proton

import (
	"context"

	"github.com/go-resty/resty/v2"
)

type OrganizationResponse struct {
	Code         int
	Organization organization
}

type organization struct {
	Name        string
	DisplayName string
	PlanName    string
	MaxMembers  int
}

func (c *Client) GetOrganizationData(ctx context.Context) (OrganizationResponse, error) {
	var res OrganizationResponse

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetResult(&res).Get("/core/v4/organizations")
	}); err != nil {
		return res, err
	}

	return res, nil
}
