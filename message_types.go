package proton

import (
	"bytes"
	"io"
	"net/mail"

	"github.com/ProtonMail/gluon/rfc822"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"golang.org/x/exp/slices"
)

type MessageMetadata struct {
	ID         string
	AddressID  string
	LabelIDs   []string
	ExternalID string

	Subject  string
	Sender   *mail.Address
	ToList   []*mail.Address
	CCList   []*mail.Address
	BCCList  []*mail.Address
	ReplyTos []*mail.Address

	Flags        MessageFlag
	Time         int64
	Size         int
	Unread       Bool
	IsReplied    Bool
	IsRepliedAll Bool
	IsForwarded  Bool
}

func (meta MessageMetadata) Seen() bool {
	return !bool(meta.Unread)
}

func (meta MessageMetadata) Starred() bool {
	return slices.Contains(meta.LabelIDs, StarredLabel)
}

func (meta MessageMetadata) IsDraft() bool {
	return meta.Flags&(MessageFlagReceived|MessageFlagSent) == 0
}

type MessageFilter struct {
	ID []string `json:",omitempty"`

	Subject    string `json:",omitempty"`
	AddressID  string `json:",omitempty"`
	ExternalID string `json:",omitempty"`
	LabelID    string `json:",omitempty"`
}

type Message struct {
	MessageMetadata

	Header        string
	ParsedHeaders Headers
	Body          string
	MIMEType      rfc822.MIMEType
	Attachments   []Attachment
}

type MessageFlag int64

const (
	MessageFlagReceived MessageFlag = 1 << iota
	MessageFlagSent
	MessageFlagInternal
	MessageFlagE2E
	MessageFlagAuto
	MessageFlagReplied
	MessageFlagRepliedAll
	MessageFlagForwarded
	MessageFlagAutoReplied
	MessageFlagImported
	MessageFlagOpened
	MessageFlagReceiptSent
	MessageFlagNotified
	MessageFlagTouched
	MessageFlagReceipt
	MessageFlagProton
	MessageFlagReceiptRequest
	MessageFlagPublicKey
	MessageFlagSign
	MessageFlagUnsubscribed
	MessageFlagSPFFail
	MessageFlagDKIMFail
	MessageFlagDMARCFail
	MessageFlagHamManual
	MessageFlagSpamAuto
	MessageFlagSpamManual
	MessageFlagPhishingAuto
	MessageFlagPhishingManual
)

func (f MessageFlag) Has(flag MessageFlag) bool {
	return f&flag != 0
}

func (f MessageFlag) Matches(flag MessageFlag) bool {
	return f&flag == flag
}

func (f MessageFlag) HasAny(flags ...MessageFlag) bool {
	for _, flag := range flags {
		if f.Has(flag) {
			return true
		}
	}

	return false
}

func (f MessageFlag) HasAll(flags ...MessageFlag) bool {
	for _, flag := range flags {
		if !f.Has(flag) {
			return false
		}
	}

	return true
}

func (f MessageFlag) Add(flag MessageFlag) MessageFlag {
	return f | flag
}

func (f MessageFlag) Remove(flag MessageFlag) MessageFlag {
	return f &^ flag
}

func (f MessageFlag) Toggle(flag MessageFlag) MessageFlag {
	if f.Has(flag) {
		return f.Remove(flag)
	}

	return f.Add(flag)
}

func (m Message) Decrypt(kr *crypto.KeyRing) ([]byte, error) {
	enc, err := crypto.NewPGPMessageFromArmored(m.Body)
	if err != nil {
		return nil, err
	}

	dec, err := kr.Decrypt(enc, nil, crypto.GetUnixTime())
	if err != nil {
		return nil, err
	}

	return dec.GetBinary(), nil
}

func (m Message) DecryptInto(kr *crypto.KeyRing, buffer io.ReaderFrom) error {
	armored, err := armor.Decode(bytes.NewReader([]byte(m.Body)))
	if err != nil {
		return err
	}

	stream, err := kr.DecryptStream(armored.Body, nil, crypto.GetUnixTime())
	if err != nil {
		return err
	}

	if _, err := buffer.ReadFrom(stream); err != nil {
		return err
	}

	return nil
}

type FullMessage struct {
	Message

	AttData [][]byte
}

type Signature struct {
	Hash string
	Data *crypto.PGPSignature
}

type MessageActionReq struct {
	IDs []string
}

type LabelMessagesReq struct {
	LabelID string
	IDs     []string
}

type LabelMessagesRes struct {
	Responses []LabelMessageRes
	UndoToken UndoToken
}

func (res LabelMessagesRes) ok() bool {
	for _, resp := range res.Responses {
		if resp.Response.Code != SuccessCode {
			return false
		}
	}

	return true
}

type LabelMessageRes struct {
	ID       string
	Response APIError
}
