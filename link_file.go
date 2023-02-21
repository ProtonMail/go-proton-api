package proton

import (
	"context"

	"github.com/go-resty/resty/v2"
)

func (c *Client) ListRevisions(ctx context.Context, shareID, linkID string) ([]RevisionMetadata, error) {
	var res struct {
		Revisions []RevisionMetadata
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetResult(&res).Get("/drive/shares/" + shareID + "/files/" + linkID + "/revisions")
	}); err != nil {
		return nil, err
	}

	return res.Revisions, nil
}

func (c *Client) GetRevision(ctx context.Context, shareID, linkID, revisionID string) (Revision, error) {
	var res struct {
		Revision Revision
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetResult(&res).Get("/drive/shares/" + shareID + "/files/" + linkID + "/revisions/" + revisionID)
	}); err != nil {
		return Revision{}, err
	}

	return res.Revision, nil
}

func (c *Client) UpdateRevision(ctx context.Context, shareID, linkID, revisionID string, req UpdateRevisionReq) error {
	return c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetBody(req).Put("/drive/shares/" + shareID + "/files/" + linkID + "/revisions/" + revisionID)
	})
}
