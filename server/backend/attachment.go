package backend

import (
	"encoding/base64"

	"github.com/ProtonMail/gluon/rfc822"
	"github.com/ProtonMail/go-proton-api"
	"github.com/google/uuid"
)

func (b *unsafeBackend) createAttData(dataPacket []byte) string {
	attDataID := uuid.NewString()

	b.attData[attDataID] = dataPacket

	return attDataID
}

type attachment struct {
	attachID  string
	attDataID string

	filename    string
	mimeType    rfc822.MIMEType
	disposition proton.Disposition
	contentID   string

	keyPackets []byte
	armSig     string
}

func newAttachment(
	filename string,
	mimeType rfc822.MIMEType,
	disposition proton.Disposition,
	contentID string,
	keyPackets []byte,
	dataPacketID string,
	armSig string,
) *attachment {
	return &attachment{
		attachID:  uuid.NewString(),
		attDataID: dataPacketID,

		filename:    filename,
		mimeType:    mimeType,
		disposition: disposition,
		contentID:   contentID,

		keyPackets: keyPackets,
		armSig:     armSig,
	}
}

func (att *attachment) toAttachment() proton.Attachment {
	return proton.Attachment{
		ID: att.attachID,

		Name:        att.filename,
		MIMEType:    att.mimeType,
		Disposition: att.disposition,

		KeyPackets: base64.StdEncoding.EncodeToString(att.keyPackets),
		Signature:  att.armSig,
	}
}
