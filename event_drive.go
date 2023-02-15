package proton

import (
	"context"

	"github.com/go-resty/resty/v2"
)

func (c *Client) GetLatestVolumeEventID(ctx context.Context, volumeID string) (string, error) {
	var res struct {
		EventID string
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetResult(&res).Get("/drive/volumes/" + volumeID + "/events/latest")
	}); err != nil {
		return "", err
	}

	return res.EventID, nil
}

func (c *Client) GetVolumeEvent(ctx context.Context, volumeID, eventID string) (VolumeEvent, error) {
	event, more, err := c.getVolumeEvent(ctx, volumeID, eventID)
	if err != nil {
		return VolumeEvent{}, err
	}

	for more {
		var next VolumeEvent

		next, more, err = c.getVolumeEvent(ctx, volumeID, event.EventID)
		if err != nil {
			return VolumeEvent{}, err
		}

		event.Events = append(event.Events, next.Events...)
	}

	return event, nil
}

func (c *Client) getVolumeEvent(ctx context.Context, volumeID, eventID string) (VolumeEvent, bool, error) {
	var res struct {
		VolumeEvent

		More Bool
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetResult(&res).Get("/drive/volumes/" + volumeID + "/events/" + eventID)
	}); err != nil {
		return VolumeEvent{}, false, err
	}

	return res.VolumeEvent, bool(res.More), nil
}
