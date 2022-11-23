package proton

type CreateAddressKeyReq struct {
	AddressID     string
	PrivateKey    string
	Primary       Bool
	SignedKeyList KeyList

	// The following are only used in "migrated accounts"
	Token     string `json:",omitempty"`
	Signature string `json:",omitempty"`
}

type MakeAddressKeyPrimaryReq struct {
	SignedKeyList KeyList
}
