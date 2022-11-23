package proton

import "github.com/ProtonMail/gopenpgp/v2/crypto"

type Volume struct {
	ID          string      // Encrypted volume ID
	Name        string      // The volume name
	OwnerUserID string      // Encrypted owner user ID
	UsedSpace   int64       // Space used by files in the volume in bytes
	MaxSpace    int64       // Space limit for the volume in bytes
	State       VolumeState // TODO: What is this?
}

type VolumeState int

const (
// TODO: VolumeState constants
)

type Share struct {
	ShareID             string      // Encrypted share ID
	Type                ShareType   // Type of share
	State               ShareState  // TODO: What is this?
	PermissionsMask     Permissions // Mask restricting member permissions on the share
	LinkID              string      // Encrypted link ID to which the share points (root of share).
	LinkType            LinkType    // TODO: What is this?
	VolumeID            string      // Encrypted volume ID on which the share is mounted
	Creator             string      // Creator address
	AddressID           string
	Flags               ShareFlags // The flag bitmap, with the following values
	BlockSize           int64      // TODO: What is this?
	Locked              bool       // TODO: What is this?
	Key                 string     // The private key, encrypted with a passphrase
	Passphrase          string     // The encrypted passphrase
	PassphraseSignature string     // The signature of the passphrase
}

func (s Share) GetKeyRing(kr *crypto.KeyRing) (*crypto.KeyRing, error) {
	encPass, err := crypto.NewPGPMessageFromArmored(s.Passphrase)
	if err != nil {
		return nil, err
	}

	decPass, err := kr.Decrypt(encPass, nil, crypto.GetUnixTime())
	if err != nil {
		return nil, err
	}

	lockedKey, err := crypto.NewKeyFromArmored(s.Key)
	if err != nil {
		return nil, err
	}

	unlockedKey, err := lockedKey.Unlock(decPass.GetBinary())
	if err != nil {
		return nil, err
	}

	return crypto.NewKeyRing(unlockedKey)
}

type ShareType int

const (
// TODO: ShareType constants
)

type ShareState int

const (
// TODO: ShareState constants
)

type ShareFlags int

const (
	NoFlags ShareFlags = iota
	PrimaryShare
)

type Link struct {
	LinkID         string // Encrypted file/folder ID
	ParentLinkID   string // Encrypted parent folder ID (LinkID)
	Type           LinkType
	Name           string    // Encrypted file name
	Hash           string    // HMAC of name encrypted with parent hash key
	State          LinkState // State of the link
	ExpirationTime int64
	Size           int64
	MIMEType       string
	Attributes     Attributes
	Permissions    Permissions

	NodeKey                 string
	NodePassphrase          string
	NodePassphraseSignature string
	SignatureAddress        string

	CreateTime int64
	ModifyTime int64

	FileProperties   FileProperties
	FolderProperties FolderProperties
}

func (l Link) GetKeyRing(kr *crypto.KeyRing) (*crypto.KeyRing, error) {
	encPass, err := crypto.NewPGPMessageFromArmored(l.NodePassphrase)
	if err != nil {
		return nil, err
	}

	decPass, err := kr.Decrypt(encPass, nil, crypto.GetUnixTime())
	if err != nil {
		return nil, err
	}

	lockedKey, err := crypto.NewKeyFromArmored(l.NodeKey)
	if err != nil {
		return nil, err
	}

	unlockedKey, err := lockedKey.Unlock(decPass.GetBinary())
	if err != nil {
		return nil, err
	}

	return crypto.NewKeyRing(unlockedKey)
}

type FileProperties struct {
	ContentKeyPacket string
	ActiveRevision   Revision
}

type FolderProperties struct{}

type LinkType int

const (
	FolderLinkType LinkType = iota + 1
	FileLinkType
)

type LinkState int

const (
	DraftLinkState LinkState = iota
	ActiveLinkState
	TrashedLinkState
	DeletedLinkState
)

type Revision struct {
	ID                string            // Encrypted Revision ID
	CreateTime        int64             //  Unix timestamp of the revision creation time
	Size              int64             //  Size of the file in bytes
	ManifestSignature string            // The signature of the root hash
	SignatureAddress  string            // The address used to sign the root hash
	State             FileRevisionState // State of revision
	Blocks            []Block
}

type FileRevisionState int

const (
	DraftRevisionState FileRevisionState = iota
	ActiveRevisionState
	ObsoleteRevisionState
)

type Block struct {
	Index          int
	URL            string
	EncSignature   string
	SignatureEmail string
}

type LinkEvent struct {
	EventID    string        // Encrypted ID of the Event
	CreateTime int64         // Time stamp of the creation time of the Event
	EventType  LinkEventType // Type of event
}

type LinkEventType int

const (
	DeleteLinkEvent LinkEventType = iota
	CreateLinkEvent
	UpdateContentsLinkEvent
	UpdateMetadataLinkEvent
)

type Permissions int

const (
	NoPermissions Permissions = 1 << iota
	ReadPermission
	WritePermission
	AdministerMembersPermission
	AdminPermission
	SuperAdminPermission
)

type Attributes uint32
