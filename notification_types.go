package proton

type NotificationPayload struct {
	Title    string
	Subtitle string
	Body     string
}

type NotificationEvent struct {
	ID      string
	UID     string
	UserID  string
	Type    string
	Time    int64
	Payload NotificationPayload
}
