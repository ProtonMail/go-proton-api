package backend

import (
	"github.com/ProtonMail/go-proton-api"
	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/emersion/go-vcard"
)

func ContactCardToContact(card *proton.Card, contactID string, kr *crypto.KeyRing) (proton.Contact, error) {
	emails, err := card.Get(kr, vcard.FieldEmail)
	if err != nil {
		return proton.Contact{}, err
	}
	names, err := card.Get(kr, vcard.FieldFormattedName)
	if err != nil {
		return proton.Contact{}, err
	}
	return proton.Contact{
		ContactMetadata: proton.ContactMetadata{
			ID:   contactID,
			Name: names[0].Value,
			ContactEmails: []proton.ContactEmail{proton.ContactEmail{
				ID:        "1",
				Name:      names[0].Value,
				Email:     emails[0].Value,
				ContactID: contactID,
			},
			},
		},
		ContactCards: proton.ContactCards{Cards: proton.Cards{card}},
	}, nil
}
