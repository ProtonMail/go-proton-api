package proton

type CreateFolderReq struct {
	ParentLinkID string

	Name string
	Hash string

	NodeKey     string
	NodeHashKey string

	NodePassphrase          string
	NodePassphraseSignature string

	SignatureEmail string
}

type CreateFolderRes struct {
	ID string // Encrypted Link ID
}
