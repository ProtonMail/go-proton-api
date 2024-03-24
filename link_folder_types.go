package proton

// CreateFolderReq contains the fields required for creating a folder. All PGP keys and signatures are in armored format.
type CreateFolderReq struct {
	ParentLinkID string // The link ID of the parent folder.

	Name string // The folder name. PGP message. encrypted using the parent folder node key, signed using the address key.
	Hash string // The folder name hash as a hex string, hashed using NodeHashKey.

	NodeKey                 string // The new node key. PGP message. Not signed.
	NodePassphrase          string // The node passphrase. PGP message. Encrypted with the parent's node key.
	NodePassphraseSignature string // The node passphrase signature. PGP message. Signed using the address key.
	NodeHashKey             string // The new node hash key .PGP message. Encrypted and signed using the node key.

	SignatureAddress string // The signature address used to sign passphrase and name.

	XAttr *string // The optional extended attributes. PGP message. Encrypted using the parent node's key. Signed using the address key.
}

// CreateFolderRes contains the result for a CreateFolder request.
type CreateFolderRes struct {
	ID string // The encrypted Link ID.
}
