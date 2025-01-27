package proton_test

import (
	"testing"

	"github.com/ProtonMail/gluon/rfc822"
	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/henrybear327/go-proton-api"
	"github.com/stretchr/testify/require"
)

func TestSendDraftReq_AddMIMEPackage(t *testing.T) {
	key, err := crypto.GenerateKey("name", "email", "rsa", 2048)
	require.NoError(t, err)

	kr, err := crypto.NewKeyRing(key)
	require.NoError(t, err)

	tests := []struct {
		name     string
		mimeBody string
		prefs    map[string]proton.SendPreferences
		wantErr  bool
	}{
		{
			name:     "Clear MIME with detached signature",
			mimeBody: "this is a mime body",
			prefs: map[string]proton.SendPreferences{"mime-sign@email.com": {
				Encrypt:          false,
				SignatureType:    proton.DetachedSignature,
				EncryptionScheme: proton.ClearMIMEScheme,
				MIMEType:         rfc822.MultipartMixed,
			}},
			wantErr: false,
		},
		{
			name:     "Clear MIME with no signature (error)",
			mimeBody: "this is a mime body",
			prefs: map[string]proton.SendPreferences{"mime-no-sign@email.com": {
				Encrypt:          false,
				SignatureType:    proton.NoSignature,
				EncryptionScheme: proton.ClearMIMEScheme,
				MIMEType:         rfc822.MultipartMixed,
			}},
			wantErr: true,
		},
		{
			name:     "Clear MIME with plain text (error)",
			mimeBody: "this is a mime body",
			prefs: map[string]proton.SendPreferences{"mime-plain@email.com": {
				Encrypt:          false,
				SignatureType:    proton.DetachedSignature,
				EncryptionScheme: proton.ClearMIMEScheme,
				MIMEType:         rfc822.TextPlain,
			}},
			wantErr: true,
		},
		{
			name:     "Clear MIME with rich text (error)",
			mimeBody: "this is a mime body",
			prefs: map[string]proton.SendPreferences{"mime-html@email.com": {
				Encrypt:          false,
				SignatureType:    proton.DetachedSignature,
				EncryptionScheme: proton.ClearMIMEScheme,
				MIMEType:         rfc822.TextHTML,
			}},
			wantErr: true,
		},
		{
			name:     "PGP MIME with detached signature",
			mimeBody: "this is a mime body",
			prefs: map[string]proton.SendPreferences{"mime-encrypted@email.com": {
				Encrypt:          true,
				PubKey:           kr,
				SignatureType:    proton.DetachedSignature,
				EncryptionScheme: proton.PGPMIMEScheme,
				MIMEType:         rfc822.MultipartMixed,
			}},
			wantErr: false,
		},
		{
			name:     "PGP MIME with plain text (error)",
			mimeBody: "this is a mime body",
			prefs: map[string]proton.SendPreferences{"mime-encrypted-plain@email.com": {
				Encrypt:          true,
				PubKey:           kr,
				SignatureType:    proton.DetachedSignature,
				EncryptionScheme: proton.PGPMIMEScheme,
				MIMEType:         rfc822.TextPlain,
			}},
			wantErr: true,
		},
		{
			name:     "PGP MIME with rich text (error)",
			mimeBody: "this is a mime body",
			prefs: map[string]proton.SendPreferences{"mime-encrypted-plain@email.com": {
				Encrypt:          true,
				PubKey:           kr,
				SignatureType:    proton.DetachedSignature,
				EncryptionScheme: proton.PGPMIMEScheme,
				MIMEType:         rfc822.TextHTML,
			}},
			wantErr: true,
		},
		{
			name:     "PGP MIME with missing public key (error)",
			mimeBody: "this is a mime body",
			prefs: map[string]proton.SendPreferences{"mime-encrypted-no-pubkey@email.com": {
				Encrypt:          true,
				SignatureType:    proton.DetachedSignature,
				EncryptionScheme: proton.PGPMIMEScheme,
				MIMEType:         rfc822.MultipartMixed,
			}},
			wantErr: true,
		},
		{
			name:     "PGP MIME with no signature (error)",
			mimeBody: "this is a mime body",
			prefs: map[string]proton.SendPreferences{"mime-encrypted-no-signature@email.com": {
				Encrypt:          true,
				PubKey:           kr,
				SignatureType:    proton.NoSignature,
				EncryptionScheme: proton.PGPMIMEScheme,
				MIMEType:         rfc822.MultipartMixed,
			}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req proton.SendDraftReq

			if err := req.AddMIMEPackage(kr, tt.mimeBody, tt.prefs); (err != nil) != tt.wantErr {
				t.Errorf("SendDraftReq.AddMIMEPackage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSendDraftReq_AddPackage(t *testing.T) {
	key, err := crypto.GenerateKey("name", "email", "rsa", 2048)
	require.NoError(t, err)

	kr, err := crypto.NewKeyRing(key)
	require.NoError(t, err)

	tests := []struct {
		name     string
		body     string
		mimeType rfc822.MIMEType
		prefs    map[string]proton.SendPreferences
		attKeys  map[string]*crypto.SessionKey
		wantErr  bool
	}{
		{
			name:     "internal plain text with detached signature",
			body:     "this is a text/plain body",
			mimeType: rfc822.TextPlain,
			prefs: map[string]proton.SendPreferences{"internal-plain@email.com": {
				Encrypt:          true,
				PubKey:           kr,
				SignatureType:    proton.DetachedSignature,
				EncryptionScheme: proton.InternalScheme,
				MIMEType:         rfc822.TextPlain,
			}},
			wantErr: false,
		},
		{
			name:     "internal rich text with detached signature",
			body:     "this is a text/html body",
			mimeType: rfc822.TextHTML,
			prefs: map[string]proton.SendPreferences{"internal-html@email.com": {
				Encrypt:          true,
				PubKey:           kr,
				SignatureType:    proton.DetachedSignature,
				EncryptionScheme: proton.InternalScheme,
				MIMEType:         rfc822.TextHTML,
			}},
			wantErr: false,
		},
		{
			name:     "internal rich text with bad package content type (error)",
			body:     "this is a text/html body",
			mimeType: "bad content type",
			prefs: map[string]proton.SendPreferences{"internal-bad-package-content-type@email.com": {
				Encrypt:          true,
				PubKey:           kr,
				SignatureType:    proton.DetachedSignature,
				EncryptionScheme: proton.InternalScheme,
				MIMEType:         rfc822.TextHTML,
			}},
			wantErr: true,
		},
		{
			name:     "internal rich text with bad recipient content type (error)",
			body:     "this is a text/html body",
			mimeType: rfc822.TextHTML,
			prefs: map[string]proton.SendPreferences{"internal-bad-recipient-content-type@email.com": {
				Encrypt:          true,
				PubKey:           kr,
				SignatureType:    proton.DetachedSignature,
				EncryptionScheme: proton.InternalScheme,
				MIMEType:         "bad content type",
			}},
			wantErr: true,
		},
		{
			name:     "internal with multipart (error)",
			body:     "this is a text/html body",
			mimeType: rfc822.MultipartMixed,
			prefs: map[string]proton.SendPreferences{"internal-multipart-mixed@email.com": {
				Encrypt:          true,
				PubKey:           kr,
				SignatureType:    proton.DetachedSignature,
				EncryptionScheme: proton.InternalScheme,
				MIMEType:         rfc822.MultipartMixed,
			}},
			wantErr: true,
		},
		{
			name:     "internal without encryption (error)",
			body:     "this is a text/html body",
			mimeType: rfc822.TextHTML,
			prefs: map[string]proton.SendPreferences{"internal-no-encrypt@email.com": {
				Encrypt:          false,
				PubKey:           kr,
				SignatureType:    proton.DetachedSignature,
				EncryptionScheme: proton.InternalScheme,
				MIMEType:         rfc822.TextHTML,
			}},
			wantErr: true,
		},
		{
			name:     "internal without pubkey (error)",
			body:     "this is a text/html body",
			mimeType: rfc822.TextHTML,
			prefs: map[string]proton.SendPreferences{"internal-no-pubkey@email.com": {
				Encrypt:          true,
				SignatureType:    proton.DetachedSignature,
				EncryptionScheme: proton.InternalScheme,
				MIMEType:         rfc822.TextHTML,
			}},
			wantErr: true,
		},
		{
			name:     "internal without signature (error)",
			body:     "this is a text/html body",
			mimeType: rfc822.TextHTML,
			prefs: map[string]proton.SendPreferences{"internal-no-sig@email.com": {
				Encrypt:          true,
				PubKey:           kr,
				SignatureType:    proton.NoSignature,
				EncryptionScheme: proton.InternalScheme,
				MIMEType:         rfc822.TextHTML,
			}},
			wantErr: true,
		},
		{
			name:     "clear rich text without signature",
			body:     "this is a text/html body",
			mimeType: rfc822.TextHTML,
			prefs: map[string]proton.SendPreferences{"clear-rich@email.com": {
				Encrypt:          false,
				SignatureType:    proton.NoSignature,
				EncryptionScheme: proton.ClearScheme,
				MIMEType:         rfc822.TextHTML,
			}},
			wantErr: false,
		},
		{
			name:     "clear plain text without signature",
			body:     "this is a text/plain body",
			mimeType: rfc822.TextPlain,
			prefs: map[string]proton.SendPreferences{"clear-plain@email.com": {
				Encrypt:          false,
				SignatureType:    proton.NoSignature,
				EncryptionScheme: proton.ClearScheme,
				MIMEType:         rfc822.TextPlain,
			}},
			wantErr: false,
		},
		{
			name:     "clear plain text with signature",
			body:     "this is a text/plain body",
			mimeType: rfc822.TextPlain,
			prefs: map[string]proton.SendPreferences{"clear-plain-with-sig@email.com": {
				Encrypt:          false,
				SignatureType:    proton.DetachedSignature,
				EncryptionScheme: proton.ClearScheme,
				MIMEType:         rfc822.TextPlain,
			}},
			wantErr: false,
		},
		{
			name:     "clear plain text with bad scheme (error)",
			body:     "this is a text/plain body",
			mimeType: rfc822.TextPlain,
			prefs: map[string]proton.SendPreferences{"clear-plain-with-sig@email.com": {
				Encrypt:          false,
				SignatureType:    proton.DetachedSignature,
				EncryptionScheme: proton.PGPInlineScheme,
				MIMEType:         rfc822.TextPlain,
			}},
			wantErr: true,
		},
		{
			name:     "clear rich text with signature (error)",
			body:     "this is a text/html body",
			mimeType: rfc822.TextHTML,
			prefs: map[string]proton.SendPreferences{"clear-plain-with-sig@email.com": {
				Encrypt:          false,
				SignatureType:    proton.DetachedSignature,
				EncryptionScheme: proton.ClearScheme,
				MIMEType:         rfc822.TextHTML,
			}},
			wantErr: true,
		},
		{
			name:     "encrypted plain text with signature",
			body:     "this is a text/plain body",
			mimeType: rfc822.TextPlain,
			prefs: map[string]proton.SendPreferences{"pgp-inline-with-sig@email.com": {
				Encrypt:          true,
				PubKey:           kr,
				SignatureType:    proton.DetachedSignature,
				EncryptionScheme: proton.PGPInlineScheme,
				MIMEType:         rfc822.TextPlain,
			}},
			wantErr: false,
		},
		{
			name:     "encrypted html text with signature (error)",
			body:     "this is a text/html body",
			mimeType: rfc822.TextHTML,
			prefs: map[string]proton.SendPreferences{"pgp-inline-rich-with-sig@email.com": {
				Encrypt:          true,
				PubKey:           kr,
				SignatureType:    proton.DetachedSignature,
				EncryptionScheme: proton.PGPInlineScheme,
				MIMEType:         rfc822.TextHTML,
			}},
			wantErr: true,
		},
		{
			name:     "encrypted mixed text with signature (error)",
			body:     "this is a multipart/mixed body",
			mimeType: rfc822.MultipartMixed,
			prefs: map[string]proton.SendPreferences{"pgp-inline-mixed-with-sig@email.com": {
				Encrypt:          true,
				PubKey:           kr,
				SignatureType:    proton.DetachedSignature,
				EncryptionScheme: proton.PGPInlineScheme,
				MIMEType:         rfc822.MultipartMixed,
			}},
			wantErr: true,
		},
		{
			name:     "encrypted for outside (error)",
			body:     "this is a text/plain body",
			mimeType: rfc822.TextPlain,
			prefs: map[string]proton.SendPreferences{"enc-for-outside@email.com": {
				Encrypt:          true,
				PubKey:           kr,
				SignatureType:    proton.DetachedSignature,
				EncryptionScheme: proton.EncryptedOutsideScheme,
				MIMEType:         rfc822.TextPlain,
			}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req proton.SendDraftReq

			if err := req.AddTextPackage(kr, tt.body, tt.mimeType, tt.prefs, tt.attKeys); (err != nil) != tt.wantErr {
				t.Errorf("SendDraftReq.AddPackage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
