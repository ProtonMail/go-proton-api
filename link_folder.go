package proton

import (
	"context"
	"fmt"

	"github.com/bradenaw/juniper/xslices"
	"github.com/go-resty/resty/v2"
)

func (c *Client) ListChildren(ctx context.Context, shareID, linkID string) ([]Link, error) {
	var res struct {
		Links []Link
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetResult(&res).Get("/drive/shares/" + shareID + "/folders/" + linkID + "/children")
	}); err != nil {
		return nil, err
	}

	return res.Links, nil
}

func (c *Client) TrashChildren(ctx context.Context, shareID, linkID string, childIDs ...string) error {
	var res struct {
		Responses struct {
			Responses []struct {
				LinkResponse struct {
					LinkID   string
					Response APIError
				}
			}
		}
	}

	for _, childIDs := range xslices.Chunk(childIDs, 150) {
		req := struct {
			LinkIDs []string
		}{
			LinkIDs: childIDs,
		}

		if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
			return r.SetBody(req).SetResult(&res).Get("/drive/shares/" + shareID + "/folders/" + linkID + "/trash_multiple")
		}); err != nil {
			return err
		}

		for _, res := range res.Responses.Responses {
			if res.LinkResponse.Response.Code != SuccessCode {
				return fmt.Errorf("failed to import message: %w", res.LinkResponse.Response)
			}
		}
	}

	return nil
}

func (c *Client) DeleteChildren(ctx context.Context, shareID, linkID string, childIDs ...string) error {
	var res struct {
		Responses struct {
			Responses []struct {
				LinkResponse struct {
					LinkID   string
					Response APIError
				}
			}
		}
	}

	for _, childIDs := range xslices.Chunk(childIDs, 150) {
		req := struct {
			LinkIDs []string
		}{
			LinkIDs: childIDs,
		}

		if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
			return r.SetBody(req).SetResult(&res).Get("/drive/shares/" + shareID + "/folders/" + linkID + "/delete_multiple")
		}); err != nil {
			return err
		}

		for _, res := range res.Responses.Responses {
			if res.LinkResponse.Response.Code != SuccessCode {
				return fmt.Errorf("failed to import message: %w", res.LinkResponse.Response)
			}
		}
	}

	return nil
}
