package proton

import (
	"bytes"
	"io"
	"mime"
	"strings"

	"github.com/ProtonMail/gluon/rfc822"
	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/google/uuid"
)

// CharsetReader returns a charset decoder for the given charset.
// If set, it will be used to decode non-utf8 encoded messages.
var CharsetReader func(charset string, input io.Reader) (io.Reader, error)

// EncryptRFC822 encrypts the given message literal as a PGP attachment.
func EncryptRFC822(kr *crypto.KeyRing, literal []byte) ([]byte, error) {
	msg, err := kr.Encrypt(crypto.NewPlainMessage(literal), kr)
	if err != nil {
		return nil, err
	}
	armored, err := msg.GetArmored()
	if err != nil {
		return nil, err
	}

	header, _ := rfc822.Split(literal)

	headerParsed, err := rfc822.NewHeader(header)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	boundary := strings.ReplaceAll(uuid.NewString(), "-", "")
	multipartWriter := rfc822.NewMultipartWriter(buf, boundary)

	{
		newHeader := rfc822.NewEmptyHeader()

		if value, ok := headerParsed.GetChecked("Message-Id"); ok {
			newHeader.Set("Message-Id", value)
		}

		contentType := mime.FormatMediaType("multipart/encrypted", map[string]string{
			"boundary": boundary,
			"protocol": "application/pgp-encrypted",
		})
		newHeader.Set("Mime-version", "1.0")
		newHeader.Set("Content-Type", contentType)

		if value, ok := headerParsed.GetChecked("From"); ok {
			newHeader.Set("From", value)
		}

		if value, ok := headerParsed.GetChecked("To"); ok {
			newHeader.Set("To", value)
		}

		if value, ok := headerParsed.GetChecked("Subject"); ok {
			newHeader.Set("Subject", value)
		}

		if value, ok := headerParsed.GetChecked("Date"); ok {
			newHeader.Set("Date", value)
		}

		if value, ok := headerParsed.GetChecked("Received"); ok {
			newHeader.Set("Received", value)
		}

		buf.Write(newHeader.Raw())
	}

	// Write PGP control data
	{
		pgpControlHeader := rfc822.NewEmptyHeader()
		pgpControlHeader.Set("Content-Description", "PGP/MIME version identification")
		pgpControlHeader.Set("Content-Type", "application/pgp-encrypted")
		if err := multipartWriter.AddPart(func(writer io.Writer) error {
			if _, err := writer.Write(pgpControlHeader.Raw()); err != nil {
				return err
			}

			_, err := writer.Write([]byte("Version: 1"))

			return err
		}); err != nil {
			return nil, err
		}
	}

	// write PGP attachment
	{
		pgpAttachmentHeader := rfc822.NewEmptyHeader()
		contentType := mime.FormatMediaType("application/octet-stream", map[string]string{
			"name": "encrypted.asc",
		})
		pgpAttachmentHeader.Set("Content-Description", "OpenPGP encrypted message")
		pgpAttachmentHeader.Set("Content-Disposition", "inline; filename=encrypted.asc")
		pgpAttachmentHeader.Set("Content-Type", contentType)

		if err := multipartWriter.AddPart(func(writer io.Writer) error {
			if _, err := writer.Write(pgpAttachmentHeader.Raw()); err != nil {
				return err
			}

			_, err := writer.Write([]byte(armored))
			return err
		}); err != nil {
			return nil, err
		}
	}

	// finish messsage
	if err := multipartWriter.Done(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
