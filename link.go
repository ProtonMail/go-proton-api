package proton

import (
	"context"

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

func (c *Client) MoveLink(ctx context.Context, shareID, linkID string, req MoveLinkReq) error {
	var res struct {
		Code int
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetResult(&res).SetBody(req).Put("/drive/shares/" + shareID + "/links/" + linkID + "/move")
	}); err != nil {
		return err
	}

	return nil
}

func (c *Client) CreateFile(ctx context.Context, shareID string, req CreateFileReq) (CreateFileRes, error) {
	var res struct {
		File CreateFileRes
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetResult(&res).SetBody(req).Post("/drive/shares/" + shareID + "/files")
	}); err != nil {
		return CreateFileRes{}, err
	}

	return res.File, nil
}

func (c *Client) CreateFolder(ctx context.Context, shareID string, req CreateFolderReq) (CreateFolderRes, error) {
	var res struct {
		Folder CreateFolderRes
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetResult(&res).SetBody(req).Post("/drive/shares/" + shareID + "/folders")
	}); err != nil {
		return CreateFolderRes{}, err
	}

	return res.Folder, nil
}
