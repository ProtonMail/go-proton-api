package proton

import (
	"fmt"
	"strings"

	"github.com/bradenaw/juniper/xslices"
)

type Event struct {
	EventID string

	Refresh RefreshFlag

	User *User

	MailSettings *MailSettings

	Messages []MessageEvent

	Labels []LabelEvent

	Addresses []AddressEvent

	UsedSpace *int
}

func (event Event) String() string {
	var parts []string

	if event.Refresh != 0 {
		parts = append(parts, fmt.Sprintf("refresh: %v", event.Refresh))
	}

	if event.User != nil {
		parts = append(parts, "user: [modified]")
	}

	if event.MailSettings != nil {
		parts = append(parts, "mail-settings: [modified]")
	}

	if len(event.Messages) > 0 {
		parts = append(parts, fmt.Sprintf(
			"messages: created=%d, updated=%d, deleted=%d",
			xslices.CountFunc(event.Messages, func(e MessageEvent) bool { return e.Action == EventCreate }),
			xslices.CountFunc(event.Messages, func(e MessageEvent) bool { return e.Action == EventUpdate || e.Action == EventUpdateFlags }),
			xslices.CountFunc(event.Messages, func(e MessageEvent) bool { return e.Action == EventDelete }),
		))
	}

	if len(event.Labels) > 0 {
		parts = append(parts, fmt.Sprintf(
			"labels: created=%d, updated=%d, deleted=%d",
			xslices.CountFunc(event.Labels, func(e LabelEvent) bool { return e.Action == EventCreate }),
			xslices.CountFunc(event.Labels, func(e LabelEvent) bool { return e.Action == EventUpdate || e.Action == EventUpdateFlags }),
			xslices.CountFunc(event.Labels, func(e LabelEvent) bool { return e.Action == EventDelete }),
		))
	}

	if len(event.Addresses) > 0 {
		parts = append(parts, fmt.Sprintf(
			"addresses: created=%d, updated=%d, deleted=%d",
			xslices.CountFunc(event.Addresses, func(e AddressEvent) bool { return e.Action == EventCreate }),
			xslices.CountFunc(event.Addresses, func(e AddressEvent) bool { return e.Action == EventUpdate || e.Action == EventUpdateFlags }),
			xslices.CountFunc(event.Addresses, func(e AddressEvent) bool { return e.Action == EventDelete }),
		))
	}

	return fmt.Sprintf("Event %s: %s", event.EventID, strings.Join(parts, ", "))
}

// merge combines this event with the other event (assumed to be newer!).
func (event *Event) merge(next Event) error {
	event.EventID = next.EventID

	if next.User != nil {
		event.User = next.User
	}

	if next.MailSettings != nil {
		event.MailSettings = next.MailSettings
	}

	for _, nextLabel := range next.Labels {
		if nextLabel.Action == EventUpdate {
			if idx := xslices.IndexFunc(event.Labels, func(old LabelEvent) bool {
				return old.Action == EventUpdate && old.ID == nextLabel.ID
			}); idx > 0 {
				event.Labels = append(xslices.Remove(event.Labels, idx, 1), nextLabel)
			}
		} else {
			event.Labels = append(event.Labels, nextLabel)
		}
	}

	for _, nextMsg := range next.Messages {
		if nextMsg.Action == EventUpdate {
			if idx := xslices.IndexFunc(event.Messages, func(old MessageEvent) bool {
				return old.Action == EventUpdate && old.ID == nextMsg.ID
			}); idx > 0 {
				event.Messages = append(xslices.Remove(event.Messages, idx, 1), nextMsg)
			}
		} else {
			event.Messages = append(event.Messages, nextMsg)
		}
	}

	for _, nextAddr := range next.Addresses {
		if nextAddr.Action == EventUpdate {
			if idx := xslices.IndexFunc(event.Addresses, func(old AddressEvent) bool {
				return old.Action == EventUpdate && old.ID == nextAddr.ID
			}); idx > 0 {
				event.Addresses = append(xslices.Remove(event.Addresses, idx, 1), nextAddr)
			}
		} else {
			event.Addresses = append(event.Addresses, nextAddr)
		}
	}

	return nil
}

type RefreshFlag uint8

const (
	RefreshMail RefreshFlag = 1 << iota   // 1<<0 = 1
	_                                     // 1<<1 = 2
	_                                     // 1<<2 = 4
	_                                     // 1<<3 = 8
	_                                     // 1<<4 = 16
	_                                     // 1<<5 = 32
	_                                     // 1<<6 = 64
	_                                     // 1<<7 = 128
	RefreshAll  RefreshFlag = 1<<iota - 1 // 1<<8 - 1 = 255
)

type EventAction int

const (
	EventDelete EventAction = iota
	EventCreate
	EventUpdate
	EventUpdateFlags
)

type EventItem struct {
	ID     string
	Action EventAction
}

type MessageEvent struct {
	EventItem

	Message MessageMetadata
}

type LabelEvent struct {
	EventItem

	Label Label
}

type AddressEvent struct {
	EventItem

	Address Address
}
