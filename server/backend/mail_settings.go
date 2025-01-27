package backend

import (
	"github.com/ProtonMail/gluon/rfc822"
	"github.com/henrybear327/go-proton-api"
)

type mailSettings struct {
	displayName   string
	sign          proton.SignExternalMessages
	pgpScheme     proton.EncryptionScheme
	draftMIMEType rfc822.MIMEType
	attachPubKey  bool
}

func newMailSettings(displayName string) *mailSettings {
	return &mailSettings{
		displayName:   displayName,
		draftMIMEType: rfc822.TextHTML,
		attachPubKey:  false,
		sign:          0,
		pgpScheme:     0,
	}
}

func (settings *mailSettings) toMailSettings() proton.MailSettings {
	return proton.MailSettings{
		DisplayName:     settings.displayName,
		DraftMIMEType:   settings.draftMIMEType,
		AttachPublicKey: proton.Bool(settings.attachPubKey),
		Sign:            settings.sign,
		PGPScheme:       settings.pgpScheme,
	}
}
