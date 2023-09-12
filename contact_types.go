package proton

import (
	"encoding/base64"
	"errors"
	"strconv"
	"strings"

	"github.com/ProtonMail/gluon/rfc822"
	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/emersion/go-vcard"
)

type RecipientType int

const (
	RecipientTypeInternal RecipientType = iota + 1
	RecipientTypeExternal
)

type ContactSettings struct {
	MIMEType *rfc822.MIMEType
	Scheme   *EncryptionScheme
	Sign     *bool
	Encrypt  *bool
	Keys     []*crypto.Key
}

type Contact struct {
	ContactMetadata
	ContactCards
}

func (cs *ContactSettings) SetMimeType(mimeType rfc822.MIMEType) {
	if cs.MIMEType == nil {
		cs.MIMEType = new(rfc822.MIMEType)
	}
	*cs.MIMEType = mimeType
}

func (cs *ContactSettings) SetScheme(scheme EncryptionScheme) {
	if cs.Scheme == nil {
		cs.Scheme = new(EncryptionScheme)
	}
	*cs.Scheme = scheme
}

func (cs *ContactSettings) SetSign(enabled bool) {
	if cs.Sign == nil {
		cs.Sign = new(bool)
	}
	*cs.Sign = enabled
}

func (cs *ContactSettings) SetEncrypt(enabled bool) {
	if cs.Encrypt == nil {
		cs.Encrypt = new(bool)
	}
	*cs.Encrypt = enabled
}

func (cs *ContactSettings) AddKey(key *crypto.Key) {
	cs.Keys = append(cs.Keys, key)
}

func (c *Contact) GetSettings(kr *crypto.KeyRing, email string, cardType CardType) (ContactSettings, error) {
	signedCard, ok := c.Cards.Get(cardType)
	if !ok {
		return ContactSettings{}, nil
	}

	group, err := signedCard.GetGroup(kr, vcard.FieldEmail, email)
	if err != nil {
		return ContactSettings{}, nil
	}

	var settings ContactSettings

	scheme, err := group.Get(FieldPMScheme)
	if err != nil {
		return ContactSettings{}, err
	}

	if len(scheme) > 0 {
		switch scheme[0] {
		case "pgp-inline":
			settings.Scheme = newPtr(PGPInlineScheme)

		case "pgp-mime":
			settings.Scheme = newPtr(PGPMIMEScheme)
		}
	}

	mimeType, err := group.Get(FieldPMMIMEType)
	if err != nil {
		return ContactSettings{}, err
	}

	if len(mimeType) > 0 {
		settings.MIMEType = newPtr(rfc822.MIMEType(mimeType[0]))
	}

	sign, err := group.Get(FieldPMSign)
	if err != nil {
		return ContactSettings{}, err
	}

	if len(sign) > 0 {
		sign, err := strconv.ParseBool(sign[0])
		if err != nil {
			return ContactSettings{}, err
		}

		settings.Sign = newPtr(sign)
	}

	encrypt, err := group.Get(FieldPMEncrypt)
	if err != nil {
		return ContactSettings{}, err
	}

	if len(encrypt) > 0 {
		encrypt, err := strconv.ParseBool(encrypt[0])
		if err != nil {
			return ContactSettings{}, err
		}

		settings.Encrypt = newPtr(encrypt)
	}

	keys, err := group.Get(vcard.FieldKey)
	if err != nil {
		return ContactSettings{}, err
	}

	if len(keys) > 0 {
		for _, key := range keys {
			dec, err := base64.StdEncoding.DecodeString(strings.SplitN(key, ",", 2)[1])
			if err != nil {
				return ContactSettings{}, err
			}

			pubKey, err := crypto.NewKey(dec)
			if err != nil {
				return ContactSettings{}, err
			}

			settings.Keys = append(settings.Keys, pubKey)
		}
	}

	return settings, nil
}

func (c *Contact) SetSettings(kr *crypto.KeyRing, email string, cardType CardType, settings ContactSettings) error {
	signedCard, ok := c.Cards.Get(cardType)
	if !ok {
		return errors.New("cannot get contact card for " + email)
	}

	group, err := signedCard.GetGroup(kr, vcard.FieldEmail, email)
	if err != nil {
		return nil
	}

	// X-PM-SCHEME
	if settings.Scheme != nil {
		switch *settings.Scheme {
		case PGPInlineScheme:
			if err := group.Set(FieldPMScheme, "pgp-inline", vcard.Params{}); err != nil {
				return err
			}
		case PGPMIMEScheme:
			if err := group.Set(FieldPMScheme, "pgp-mime", vcard.Params{}); err != nil {
				return err
			}
		}
	}
	// X-PM-MIMETYPE
	if settings.MIMEType != nil {
		switch *settings.MIMEType {
		case rfc822.TextPlain:
			if err := group.Set(FieldPMMIMEType, string(rfc822.TextPlain), vcard.Params{}); err != nil {
				return err
			}
		case rfc822.TextHTML:
			if err := group.Set(FieldPMMIMEType, string(rfc822.TextHTML), vcard.Params{}); err != nil {
				return err
			}
		case rfc822.MultipartMixed:
			if err := group.Set(FieldPMMIMEType, string(rfc822.MultipartMixed), vcard.Params{}); err != nil {
				return err
			}
		case rfc822.MultipartRelated:
			if err := group.Set(FieldPMMIMEType, string(rfc822.MultipartRelated), vcard.Params{}); err != nil {
				return err
			}
		case rfc822.MessageRFC822:
			if err := group.Set(FieldPMMIMEType, string(rfc822.MessageRFC822), vcard.Params{}); err != nil {
				return err
			}
		}
	}
	// X-PM-SIGN
	if settings.Sign != nil {
		if *settings.Sign {
			if err := group.Set(FieldPMSign, "true", vcard.Params{}); err != nil {
				return err
			}
		} else {
			if err := group.Set(FieldPMSign, "false", vcard.Params{}); err != nil {
				return err
			}
		}
	}

	// X-PM-ENCRYPT
	if settings.Encrypt != nil {
		if *settings.Encrypt {
			if err := group.Set(FieldPMEncrypt, "true", vcard.Params{}); err != nil {
				return err
			}
		} else {
			if err := group.Set(FieldPMEncrypt, "false", vcard.Params{}); err != nil {
				return err
			}
		}
	}

	// KEY
	if settings.Keys != nil && len(settings.Keys) > 0 {
		var keys = ""
		for i, key := range settings.Keys {
			if i > 0 {
				keys += ","
			}
			if dec, err := key.Serialize(); err == nil {
				keys += string(dec)
			}
		}
		enc := base64.StdEncoding.EncodeToString([]byte(keys))
		if err := group.Set(vcard.FieldKey, "base64,"+enc, vcard.Params{}); err != nil {
			return err
		}
	}
	*signedCard = group.Card
	return nil
}

type ContactMetadata struct {
	ID            string
	Name          string
	UID           string
	Size          int64
	CreateTime    int64
	ModifyTime    int64
	ContactEmails []ContactEmail
	LabelIDs      []string
}

type ContactCards struct {
	Cards Cards
}

type ContactEmail struct {
	ID        string
	Name      string
	Email     string
	Type      []string
	ContactID string
	LabelIDs  []string
}

type CreateContactsReq struct {
	Contacts  []ContactCards
	Overwrite int
	Labels    int
}

type CreateContactResp struct {
	APIError
	Contact Contact
}

type CreateContactsRes struct {
	Index int

	Response CreateContactResp
}

type UpdateContactReq struct {
	Cards Cards
}

type DeleteContactsReq struct {
	IDs []string
}

func newPtr[T any](v T) *T {
	return &v
}
