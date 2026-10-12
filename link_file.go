package proton

import (
	"context"
	"fmt"
	"strconv"

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

func (c *Client) GetRevision(ctx context.Context, shareID, linkID, revisionID string, fromBlock, pageSize int) (Revision, error) {
	if fromBlock < 1 {
		return Revision{}, fmt.Errorf("fromBlock must be greater than 0")
	} else if pageSize < 1 {
		return Revision{}, fmt.Errorf("pageSize must be greater than 0")
	}

	var res struct {
		Revision Revision
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.
			SetQueryParams(map[string]string{
				"FromBlockIndex": strconv.Itoa(fromBlock),
				"PageSize":       strconv.Itoa(pageSize),
			}).
			SetResult(&res).
			Get("/drive/shares/" + shareID + "/files/" + linkID + "/revisions/" + revisionID)
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

// VerificationData holds the server-provided verification code and
// content key packet for block upload verification.
type VerificationData struct {
	VerificationCode string
	ContentKeyPacket string
}

// GetVerificationData fetches the verification code for a revision.
// The verification code is used to compute per-block verification tokens.
func (c *Client) GetVerificationData(ctx context.Context, shareID, linkID, revisionID string) (VerificationData, error) {
	var res VerificationData

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetResult(&res).Get("/drive/shares/" + shareID + "/links/" + linkID + "/revisions/" + revisionID + "/verification")
	}); err != nil {
		return VerificationData{}, err
	}

	return res, nil
}
