package proton

import (
	"context"
	"time"

	"github.com/ProtonMail/gluon/async"
	"github.com/go-resty/resty/v2"
)

func (c *Client) GetLatestEventID(ctx context.Context) (string, error) {
	var res struct {
		Event
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetResult(&res).Get("/core/v4/events/latest")
	}); err != nil {
		return "", err
	}

	return res.EventID, nil
}

// maxMergedEvents limits the number of events which are merged per one GetEvent
// call.
const maxMergedEvents = 50

func (c *Client) GetEvent(ctx context.Context, eventID string) (Event, bool, error) {
	event, more, err := c.getEvent(ctx, eventID)
	if err != nil {
		return Event{}, more, err
	}

	nMerged := 0

	for more {
		nMerged++
		if nMerged >= maxMergedEvents {
			break
		}

		var next Event

		next, more, err = c.getEvent(ctx, event.EventID)
		if err != nil {
			return Event{}, false, err
		}

		if err := event.merge(next); err != nil {
			return Event{}, false, err
		}
	}

	return event, more, nil
}

// NewEventStreamer returns a new event stream.
// It polls the API for new events at random intervals between `period` and `period+jitter`.
func (c *Client) NewEventStream(ctx context.Context, period, jitter time.Duration, lastEventID string) <-chan Event {
	eventCh := make(chan Event)

	go func() {
		defer async.HandlePanic(c.m.panicHandler)

		defer close(eventCh)

		ticker := NewTicker(period, jitter, c.m.panicHandler)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				// ...
			}

			event, _, err := c.GetEvent(ctx, lastEventID)
			if err != nil {
				continue
			}

			if event.EventID == lastEventID {
				continue
			}

			select {
			case <-ctx.Done():
				return

			case eventCh <- event:
				lastEventID = event.EventID
			}
		}
	}()

	return eventCh
}

func (c *Client) getEvent(ctx context.Context, eventID string) (Event, bool, error) {
	var res struct {
		Event

		More Bool
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetResult(&res).Get("/core/v4/events/" + eventID)
	}); err != nil {
		return Event{}, false, err
	}

	return res.Event, bool(res.More), nil
}
