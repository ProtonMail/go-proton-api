package proton

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"

	"github.com/ProtonMail/gopenpgp/v2/crypto"
)

// getEncryptedName encrypts name with the node keyring, signs it with the
// address keyring, and returns the armored ciphertext.
func getEncryptedName(name string, addrKR, nodeKR *crypto.KeyRing) (string, error) {
	clearTextName := crypto.NewPlainMessageFromString(name)

	encName, err := nodeKR.Encrypt(clearTextName, addrKR)
	if err != nil {
		return "", err
	}

	encNameString, err := encName.GetArmored()
	if err != nil {
		return "", err
	}

	return encNameString, nil
}

// GetNameHash returns the hex-encoded HMAC-SHA256 of name keyed by hashKey.
func GetNameHash(name string, hashKey []byte) (string, error) {
	mac := hmac.New(sha256.New, hashKey)
	_, err := mac.Write([]byte(name))
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(mac.Sum(nil)), nil
}

type CreateFileReq struct {
	ParentLinkID string

	Name     string // Encrypted File Name
	Hash     string // Encrypted content hash
	MIMEType string // MIME Type

	ContentKeyPacket          string // The block's key packet, encrypted with the node key.
	ContentKeyPacketSignature string // Unencrypted signature of the content session key, signed with the NodeKey

	NodeKey                 string // The private NodeKey, used to decrypt any file/folder content.
	NodePassphrase          string // The passphrase used to unlock the NodeKey, encrypted by the owning Link/Share keyring.
	NodePassphraseSignature string // The signature of the NodePassphrase

	SignatureAddress string // Signature email address used to sign passphrase and name
}

func (createFileReq *CreateFileReq) SetName(name string, addrKR, nodeKR *crypto.KeyRing) error {
	encNameString, err := getEncryptedName(name, addrKR, nodeKR)
	if err != nil {
		return err
	}

	createFileReq.Name = encNameString
	return nil
}

func (createFileReq *CreateFileReq) SetHash(name string, hashKey []byte) error {
	nameHash, err := GetNameHash(name, hashKey)
	if err != nil {
		return err
	}

	createFileReq.Hash = nameHash

	return nil
}

func (createFileReq *CreateFileReq) SetContentKeyPacketAndSignature(kr *crypto.KeyRing) (*crypto.SessionKey, error) {
	newSessionKey, err := crypto.GenerateSessionKey()
	if err != nil {
		return nil, err
	}

	encSessionKey, err := kr.EncryptSessionKey(newSessionKey)
	if err != nil {
		return nil, err
	}

	sessionKeyPlainMessage := crypto.NewPlainMessage(newSessionKey.Key)
	sessionKeySignature, err := kr.SignDetached(sessionKeyPlainMessage)
	if err != nil {
		return nil, err
	}
	armoredSessionKeySignature, err := sessionKeySignature.GetArmored()
	if err != nil {
		return nil, err
	}

	createFileReq.ContentKeyPacket = base64.StdEncoding.EncodeToString(encSessionKey)
	createFileReq.ContentKeyPacketSignature = armoredSessionKeySignature
	return newSessionKey, nil
}

type CreateFileRes struct {
	ID         string // Encrypted Link ID
	RevisionID string // Encrypted Revision ID
}

type UpdateRevisionReq struct {
	BlockList         []BlockToken
	State             RevisionState
	ManifestSignature string
	SignatureAddress  string
}

type BlockToken struct {
	Index int
	Token string
}
