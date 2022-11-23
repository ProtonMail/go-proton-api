package proton

import (
	"context"

	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/go-resty/resty/v2"
)

func (c *Client) GetLink(ctx context.Context, shareID, linkID string) (Link, error) {
	var res struct {
		Link Link
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetResult(&res).Get("/drive/shares/" + shareID + "/links/" + linkID)
	}); err != nil {
		return Link{}, err
	}

	return res.Link, nil
}

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

func (c *Client) ListRevisions(ctx context.Context, shareID, linkID string) ([]Revision, error) {
	var res struct {
		Revisions []Revision
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

func (c *Client) VisitLink(ctx context.Context, shareID string, link Link, kr *crypto.KeyRing, fn LinkWalkFunc) error {
	return c.visitLink(ctx, shareID, link, kr, fn, []string{})
}

func (c *Client) visitLink(ctx context.Context, shareID string, link Link, kr *crypto.KeyRing, fn LinkWalkFunc, path []string) error {
	enc, err := crypto.NewPGPMessageFromArmored(link.Name)
	if err != nil {
		return err
	}

	dec, err := kr.Decrypt(enc, nil, crypto.GetUnixTime())
	if err != nil {
		return err
	}

	path = append(path, dec.GetString())

	childKR, err := link.GetKeyRing(kr)
	if err != nil {
		return err
	}

	if err := fn(path, link, childKR); err != nil {
		return err
	}

	if link.Type != FolderLinkType {
		return nil
	}

	children, err := c.ListChildren(ctx, shareID, link.LinkID)
	if err != nil {
		return err
	}

	for _, child := range children {
		if err := c.visitLink(ctx, shareID, child, childKR, fn, path); err != nil {
			return err
		}
	}

	return nil
}
