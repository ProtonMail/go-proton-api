package proton

import (
	"crypto/hmac"
	"crypto/sha256"
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

type MoveLinkReq struct {
	ParentLinkID string

	Name                    string // Encrypted File Name
	OriginalHash            string // Old Encrypted File Name Hash
	Hash                    string // Encrypted File Name Hash by using parent's NodeHashKey
	NodePassphrase          string // The passphrase used to unlock the NodeKey, encrypted by the owning Link/Share keyring.
	NodePassphraseSignature string // The signature of the NodePassphrase

	SignatureAddress string // Signature email address used to sign passphrase and name
}

func (moveLinkReq *MoveLinkReq) SetName(name string, addrKR, nodeKR *crypto.KeyRing) error {
	encNameString, err := getEncryptedName(name, addrKR, nodeKR)
	if err != nil {
		return err
	}

	moveLinkReq.Name = encNameString
	return nil
}

func (moveLinkReq *MoveLinkReq) SetHash(name string, hashKey []byte) error {
	nameHash, err := GetNameHash(name, hashKey)
	if err != nil {
		return err
	}

	moveLinkReq.Hash = nameHash
	return nil
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
