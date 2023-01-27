package proton

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"strconv"

	"github.com/bradenaw/juniper/parallel"
	"github.com/bradenaw/juniper/xslices"
	"github.com/go-resty/resty/v2"
)

const maxMessageIDs = 1000

func (c *Client) GetFullMessage(ctx context.Context, messageID string, scheduler Scheduler, storageProvider AttachmentAllocator) (FullMessage, error) {
	message, err := c.GetMessage(ctx, messageID)
	if err != nil {
		return FullMessage{}, err
	}

	attDataBuffers, err := scheduler.Schedule(ctx, xslices.Map(message.Attachments, func(att Attachment) string {
		return att.ID
	}), storageProvider, func(s string, buffer *bytes.Buffer) error {
		return c.GetAttachmentInto(ctx, s, buffer)
	})
	if err != nil {
		return FullMessage{}, err
	}

	return FullMessage{
		Message:        message,
		AttDataBuffers: attDataBuffers,
		AttData: xslices.Map(attDataBuffers, func(b *bytes.Buffer) []byte {
			return b.Bytes()
		}),
	}, nil
}

func (c *Client) GetMessage(ctx context.Context, messageID string) (Message, error) {
	var res struct {
		Message Message
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetResult(&res).Get("/mail/v4/messages/" + messageID)
	}); err != nil {
		return Message{}, err
	}

	return res.Message, nil
}

func (c *Client) CountMessages(ctx context.Context) (int, error) {
	return c.countMessages(ctx, MessageFilter{})
}

func (c *Client) GetMessageMetadata(ctx context.Context, filter MessageFilter) ([]MessageMetadata, error) {
	count, err := c.countMessages(ctx, filter)
	if err != nil {
		return nil, err
	}

	return fetchPaged(ctx, count, maxPageSize, func(ctx context.Context, page, pageSize int) ([]MessageMetadata, error) {
		return c.GetMessageMetadataPaged(ctx, page, pageSize, filter)
	})
}

func (c *Client) GetMessageIDs(ctx context.Context, afterID string) ([]string, error) {
	var messageIDs []string

	for ; ; afterID = messageIDs[len(messageIDs)-1] {
		page, err := c.getMessageIDs(ctx, afterID)
		if err != nil {
			return nil, err
		}

		if len(page) == 0 {
			return messageIDs, nil
		}

		messageIDs = append(messageIDs, page...)
	}
}

func (c *Client) DeleteMessage(ctx context.Context, messageIDs ...string) error {
	pages := xslices.Chunk(messageIDs, maxPageSize)

	return parallel.DoContext(ctx, runtime.NumCPU(), len(pages), func(ctx context.Context, idx int) error {
		return c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
			return r.SetBody(MessageActionReq{IDs: pages[idx]}).Put("/mail/v4/messages/delete")
		})
	})
}

func (c *Client) MarkMessagesRead(ctx context.Context, messageIDs ...string) error {
	pages := xslices.Chunk(messageIDs, maxPageSize)

	return parallel.DoContext(ctx, runtime.NumCPU(), len(pages), func(ctx context.Context, idx int) error {
		return c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
			return r.SetBody(MessageActionReq{IDs: pages[idx]}).Put("/mail/v4/messages/read")
		})
	})
}

func (c *Client) MarkMessagesUnread(ctx context.Context, messageIDs ...string) error {
	pages := xslices.Chunk(messageIDs, maxPageSize)

	return parallel.DoContext(ctx, runtime.NumCPU(), len(pages), func(ctx context.Context, idx int) error {
		req := MessageActionReq{IDs: pages[idx]}

		if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
			return r.SetBody(req).Put("/mail/v4/messages/unread")
		}); err != nil {
			return err
		}

		return nil
	})
}

func (c *Client) LabelMessages(ctx context.Context, messageIDs []string, labelID string) error {
	res, err := parallel.MapContext(
		ctx,
		runtime.NumCPU(),
		xslices.Chunk(messageIDs, maxPageSize),
		func(ctx context.Context, messageIDs []string) (LabelMessagesRes, error) {
			var res LabelMessagesRes

			if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
				return r.SetBody(LabelMessagesReq{
					LabelID: labelID,
					IDs:     messageIDs,
				}).SetResult(&res).Put("/mail/v4/messages/label")
			}); err != nil {
				return LabelMessagesRes{}, err
			}

			return res, nil
		},
	)
	if err != nil {
		return err
	}

	if idx := xslices.IndexFunc(res, func(res LabelMessagesRes) bool { return !res.ok() }); idx >= 0 {
		tokens := xslices.Map(res, func(res LabelMessagesRes) UndoToken {
			return res.UndoToken
		})

		if _, undoErr := c.UndoActions(ctx, tokens...); undoErr != nil {
			return fmt.Errorf("failed to undo actions: %w", undoErr)
		}

		return fmt.Errorf("failed to label messages")
	}

	return nil
}

func (c *Client) UnlabelMessages(ctx context.Context, messageIDs []string, labelID string) error {
	res, err := parallel.MapContext(
		ctx,
		runtime.NumCPU(),
		xslices.Chunk(messageIDs, maxPageSize),
		func(ctx context.Context, messageIDs []string) (LabelMessagesRes, error) {
			var res LabelMessagesRes

			if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
				return r.SetBody(LabelMessagesReq{
					LabelID: labelID,
					IDs:     messageIDs,
				}).SetResult(&res).Put("/mail/v4/messages/unlabel")
			}); err != nil {
				return LabelMessagesRes{}, err
			}

			return res, nil
		},
	)
	if err != nil {
		return err
	}

	if idx := xslices.IndexFunc(res, func(res LabelMessagesRes) bool { return !res.ok() }); idx >= 0 {
		tokens := xslices.Map(res, func(res LabelMessagesRes) UndoToken {
			return res.UndoToken
		})

		if _, undoErr := c.UndoActions(ctx, tokens...); undoErr != nil {
			return fmt.Errorf("failed to undo actions: %w", undoErr)
		}

		return fmt.Errorf("failed to unlabel messages")
	}

	return nil
}

func (c *Client) getMessageIDs(ctx context.Context, afterID string) ([]string, error) {
	var res struct {
		IDs []string
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		if afterID != "" {
			r = r.SetQueryParam("AfterID", afterID)
		}

		return r.SetQueryParam("Limit", strconv.Itoa(maxMessageIDs)).SetResult(&res).Get("/mail/v4/messages/ids")
	}); err != nil {
		return nil, err
	}

	return res.IDs, nil
}

func (c *Client) countMessages(ctx context.Context, filter MessageFilter) (int, error) {
	var res struct {
		Total int
	}

	req := struct {
		MessageFilter

		Limit int `json:",,string"`
	}{
		MessageFilter: filter,

		Limit: 0,
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetBody(req).SetResult(&res).SetHeader("X-HTTP-Method-Override", "GET").Post("/mail/v4/messages")
	}); err != nil {
		return 0, err
	}

	return res.Total, nil
}

func (c *Client) GetMessageMetadataPaged(ctx context.Context, page, pageSize int, filter MessageFilter) ([]MessageMetadata, error) {
	var res struct {
		Messages []MessageMetadata
		Stale    Bool
	}

	req := struct {
		MessageFilter

		Page     int
		PageSize int

		Sort string
		Desc Bool
	}{
		MessageFilter: filter,

		Page:     page,
		PageSize: pageSize,

		Sort: "ID",
		Desc: false,
	}

	for {
		if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
			return r.SetBody(req).SetResult(&res).SetHeader("X-HTTP-Method-Override", "GET").Post("/mail/v4/messages")
		}); err != nil {
			return nil, err
		}

		if !res.Stale {
			break
		}
	}

	return res.Messages, nil
}
