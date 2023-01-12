package backend

import (
	"net/mail"
	"strings"
	"time"

	"github.com/ProtonMail/gluon/rfc822"
	"github.com/ProtonMail/go-proton-api"
	"github.com/bradenaw/juniper/xslices"
	"github.com/google/uuid"
	"golang.org/x/exp/slices"
)

type message struct {
	messageID  string
	externalID string
	addrID     string
	labelIDs   []string
	sysLabel   *string
	attIDs     []string

	subject  string
	sender   *mail.Address
	toList   []*mail.Address
	ccList   []*mail.Address
	bccList  []*mail.Address
	replytos []*mail.Address
	date     time.Time

	armBody  string
	mimeType rfc822.MIMEType

	flags   proton.MessageFlag
	unread  bool
	starred bool
}

func newMessage(
	addrID string,
	subject string,
	sender *mail.Address,
	toList, ccList, bccList, replytos []*mail.Address,
	armBody string,
	mimeType rfc822.MIMEType,
	externalID string,
	date time.Time,
) *message {
	return &message{
		messageID:  uuid.NewString(),
		externalID: externalID,
		addrID:     addrID,
		sysLabel:   pointer(""),

		subject:  subject,
		sender:   sender,
		toList:   toList,
		ccList:   ccList,
		bccList:  bccList,
		replytos: replytos,
		date:     date,

		armBody:  armBody,
		mimeType: mimeType,
	}
}

func newMessageFromTemplate(addrID string, template proton.DraftTemplate) *message {
	return &message{
		messageID:  uuid.NewString(),
		externalID: template.ExternalID,
		addrID:     addrID,
		sysLabel:   pointer(""),

		subject: template.Subject,
		sender:  template.Sender,
		toList:  template.ToList,
		ccList:  template.CCList,
		bccList: template.BCCList,
		unread:  bool(template.Unread),

		armBody:  template.Body,
		mimeType: template.MIMEType,
	}
}

func (msg *message) toMessage(att map[string]*attachment) proton.Message {
	return proton.Message{
		MessageMetadata: msg.toMetadata(),

		Header:        msg.getHeader(),
		ParsedHeaders: msg.getParsedHeaders(),
		Body:          msg.armBody,
		MIMEType:      msg.mimeType,
		Attachments: xslices.Map(msg.attIDs, func(attID string) proton.Attachment {
			return att[attID].toAttachment()
		}),
	}
}

func (msg *message) toMetadata() proton.MessageMetadata {
	labelIDs := []string{proton.AllMailLabel}

	if msg.flags.Has(proton.MessageFlagSent) {
		labelIDs = append(labelIDs, proton.AllSentLabel)
	}

	if !msg.flags.HasAny(proton.MessageFlagSent, proton.MessageFlagReceived) {
		labelIDs = append(labelIDs, proton.AllDraftsLabel)
	}

	if msg.starred {
		labelIDs = append(labelIDs, proton.StarredLabel)
	}

	if msg.sysLabel != nil {
		if *msg.sysLabel != "" {
			labelIDs = append(labelIDs, *msg.sysLabel)
		}
	} else {
		switch {
		case msg.flags.Has(proton.MessageFlagReceived):
			labelIDs = append(labelIDs, proton.InboxLabel)

		case msg.flags.Has(proton.MessageFlagSent):
			labelIDs = append(labelIDs, proton.SentLabel)

		default:
			labelIDs = append(labelIDs, proton.DraftsLabel)
		}
	}

	return proton.MessageMetadata{
		ID:         msg.messageID,
		ExternalID: msg.externalID,
		AddressID:  msg.addrID,
		LabelIDs:   append(msg.labelIDs, labelIDs...),

		Subject:  msg.subject,
		Sender:   msg.sender,
		ToList:   msg.toList,
		CCList:   msg.ccList,
		BCCList:  msg.bccList,
		ReplyTos: msg.replytos,

		Flags:  msg.flags,
		Unread: proton.Bool(msg.unread),
	}
}

func (msg *message) getHeader() string {
	builder := new(strings.Builder)

	builder.WriteString("Subject: " + msg.subject + "\r\n")

	if msg.sender != nil && (msg.sender.Name != "" || msg.sender.Address != "") {
		builder.WriteString("From: " + msg.sender.String() + "\r\n")
	}

	if len(msg.toList) > 0 {
		builder.WriteString("To: " + toAddressList(msg.toList) + "\r\n")
	}

	if len(msg.ccList) > 0 {
		builder.WriteString("Cc: " + toAddressList(msg.ccList) + "\r\n")
	}

	if len(msg.bccList) > 0 {
		builder.WriteString("Bcc: " + toAddressList(msg.bccList) + "\r\n")
	}

	if msg.mimeType != "" {
		builder.WriteString("Content-Type: " + string(msg.mimeType) + "\r\n")
	}

	builder.WriteString("Date: " + msg.date.Format(time.RFC822) + "\r\n")

	return builder.String()
}

func (msg *message) getParsedHeaders() proton.Headers {
	header, err := rfc822.NewHeader([]byte(msg.getHeader()))
	if err != nil {
		panic(err)
	}

	parsed := make(proton.Headers)

	header.Entries(func(key, value string) {
		parsed[key] = append(parsed[key], value)
	})

	return parsed
}

// applyChanges will apply non-nil field from passed message.
//
// NOTE: This is not feature complete. It might panic on non-implemented changes.
func (msg *message) applyChanges(changes proton.DraftTemplate) {
	if changes.Subject != "" {
		msg.subject = changes.Subject
	}

	if changes.Sender != nil {
		msg.sender = changes.Sender
	}

	if changes.ToList != nil {
		msg.toList = append([]*mail.Address{}, changes.ToList...)
	}

	if changes.CCList != nil {
		msg.ccList = append([]*mail.Address{}, changes.CCList...)
	}

	if changes.BCCList != nil {
		msg.bccList = append([]*mail.Address{}, changes.BCCList...)
	}

	if changes.Body != "" {
		msg.armBody = changes.Body
	}

	if changes.MIMEType != "" {
		msg.mimeType = changes.MIMEType
	}

	if changes.ExternalID != "" {
		msg.externalID = changes.ExternalID
	}
}

func (msg *message) addLabel(labelID string, labels map[string]*label) {
	switch labelID {
	case proton.InboxLabel, proton.SentLabel, proton.DraftsLabel:
		msg.addFlagLabel(labelID, labels)

	case proton.TrashLabel, proton.SpamLabel, proton.ArchiveLabel:
		msg.addSystemLabel(labelID, labels)

	case proton.StarredLabel:
		msg.starred = true

	default:
		if label, ok := labels[labelID]; ok {
			msg.addUserLabel(label, labels)
		}
	}
}

func (msg *message) addFlagLabel(labelID string, labels map[string]*label) {
	msg.labelIDs = xslices.Filter(msg.labelIDs, func(otherLabelID string) bool {
		return labels[otherLabelID].labelType == proton.LabelTypeLabel
	})

	msg.sysLabel = nil
}

func (msg *message) addSystemLabel(labelID string, labels map[string]*label) {
	msg.labelIDs = xslices.Filter(msg.labelIDs, func(otherLabelID string) bool {
		return labels[otherLabelID].labelType == proton.LabelTypeLabel
	})

	msg.sysLabel = &labelID
}

func (msg *message) addUserLabel(label *label, labels map[string]*label) {
	if label.labelType != proton.LabelTypeLabel {
		msg.labelIDs = xslices.Filter(msg.labelIDs, func(otherLabelID string) bool {
			return labels[otherLabelID].labelType == proton.LabelTypeLabel
		})

		msg.sysLabel = pointer("")
	}

	if !slices.Contains(msg.labelIDs, label.labelID) {
		msg.labelIDs = append(msg.labelIDs, label.labelID)
	}
}

func (msg *message) remLabel(labelID string, labels map[string]*label) {
	switch labelID {
	case proton.InboxLabel, proton.SentLabel, proton.DraftsLabel:
		msg.remFlagLabel(labelID, labels)

	case proton.TrashLabel, proton.SpamLabel, proton.ArchiveLabel:
		msg.remSystemLabel(labelID, labels)

	case proton.StarredLabel:
		msg.starred = false

	default:
		if label, ok := labels[labelID]; ok {
			msg.remUserLabel(label, labels)
		}
	}
}

func (msg *message) remFlagLabel(labelID string, labels map[string]*label) {
	msg.sysLabel = pointer("")
}

func (msg *message) remSystemLabel(labelID string, labels map[string]*label) {
	if msg.sysLabel != nil && *msg.sysLabel == labelID {
		msg.sysLabel = pointer("")
	}
}

func (msg *message) remUserLabel(label *label, labels map[string]*label) {
	msg.labelIDs = xslices.Filter(msg.labelIDs, func(otherLabelID string) bool {
		return otherLabelID != label.labelID
	})
}

func toAddressList(addrs []*mail.Address) string {
	res := make([]string, len(addrs))

	for i, addr := range addrs {
		res[i] = addr.String()
	}

	return strings.Join(res, ", ")
}

func pointer[T any](v T) *T {
	return &v
}
