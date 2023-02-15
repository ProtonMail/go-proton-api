package proton

type VolumeEvent struct {
	EventID string

	Events []LinkEvent

	Refresh Bool
}

type LinkEvent struct {
	EventID string

	EventType LinkEventType

	CreateTime int

	Data string

	Link Link
}

type LinkEventType int

const (
	LinkEventDelete LinkEventType = iota
	LinkEventCreate
	LinkEventUpdate
	LinkEventUpdateMetadata
)
