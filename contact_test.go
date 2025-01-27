package proton_test

import (
	"bytes"
	"testing"

	"github.com/ProtonMail/gluon/rfc822"
	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/emersion/go-vcard"
	"github.com/henrybear327/go-proton-api"
	"github.com/stretchr/testify/require"
)

const message = `From: Nathaniel Borenstein <nsb@bellcore.com>
To:  Ned Freed <ned@innosoft.com>
Subject: Sample message (import 2)
MIME-Version: 1.0
Content-type: text/plain

This is explicitly typed plain ASCII text.
`

func TestContactSettings(t *testing.T) {
	card, err := proton.NewCard(nil, proton.CardTypeClear)
	require.NoError(t, err)
	var field = vcard.Field{Value: "user@user"}
	err = card.Set(nil, vcard.FieldEmail, &field)
	require.NoError(t, err)
	var contact = proton.Contact{
		ContactMetadata: proton.ContactMetadata{},
		ContactCards: proton.ContactCards{
			Cards: []*proton.Card{card},
		},
	}
	settings, err := contact.GetSettings(nil, "user@user", proton.CardTypeClear)
	require.NoError(t, err)

	require.Equal(t, settings.MIMEType, (*rfc822.MIMEType)(nil))
	require.Equal(t, settings.Scheme, (*proton.EncryptionScheme)(nil))
	require.Equal(t, settings.Sign, (*bool)(nil))
	require.Equal(t, settings.Encrypt, (*bool)(nil))
	require.Equal(t, settings.Keys, ([]*crypto.Key)(nil))

	key, err := crypto.GenerateKey("user", "user@user", "x25519", 0)
	require.NoError(t, err)
	encryptedMessage, err := encryptMessage(key)
	require.NoError(t, err)

	settings.SetMimeType(rfc822.TextPlain)
	settings.SetScheme(proton.PGPInlineScheme)
	settings.SetSign(true)
	settings.SetEncrypt(true)
	settings.SetEncryptUntrusted(true)
	settings.AddKey(key)

	err = contact.SetSettings(nil, "user@user", proton.CardTypeClear, settings)
	require.NoError(t, err)

	settings, err = contact.GetSettings(nil, "user@user", proton.CardTypeClear)
	require.NoError(t, err)

	require.Equal(t, *settings.MIMEType, rfc822.TextPlain)
	require.Equal(t, *settings.Scheme, proton.PGPInlineScheme)
	require.Equal(t, *settings.Sign, true)
	require.Equal(t, *settings.Encrypt, true)
	require.Equal(t, *settings.EncryptUntrusted, true)
	require.Equal(t, len(settings.Keys), 1)
	kr, err := crypto.NewKeyRing(settings.Keys[0])
	require.NoError(t, err)

	// check the key
	dec, err := decryptBody(kr, encryptedMessage)
	require.NoError(t, err)
	require.Equal(t, "This is explicitly typed plain ASCII text.\n", dec.GetString())
}

func encryptMessage(key *crypto.Key) ([]byte, error) {
	var buf bytes.Buffer
	kr, err := crypto.NewKeyRing(key)
	if err != nil {
		return buf.Bytes(), err
	}
	return proton.EncryptRFC822(kr, []byte(message))
}

func decryptBody(kr *crypto.KeyRing, encryptedMessage []byte) (*crypto.PlainMessage, error) {
	section := rfc822.Parse(encryptedMessage)
	// Read the body.
	body, err := section.DecodedBody()
	if err != nil {
		return nil, err
	}
	enc, err := crypto.NewPGPMessageFromArmored(string(body))
	if err != nil {
		return nil, err
	}
	return kr.Decrypt(enc, nil, crypto.GetUnixTime())
}
