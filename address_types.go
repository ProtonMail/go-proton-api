package proton

type Address struct {
	ID    string
	Email string

	Send    Bool
	Receive Bool
	Status  AddressStatus
	Type    AddressType

	Order       int
	DisplayName string

	Keys Keys
}

type OrderAddressesReq struct {
	AddressIDs []string
}

type AddressStatus int

const (
	AddressStatusDisabled AddressStatus = iota
	AddressStatusEnabled
	AddressStatusDeleting
)

type AddressType int

const (
	AddressTypeOriginal AddressType = iota + 1
	AddressTypeAlias
	AddressTypeCustom
	AddressTypePremium
	AddressTypeExternal
)

// IsBYOEAddress - return a bool corresponding to whether an address is a BYOE address.
// BYOE addresses have sending enabled and are of type `external`.
func (a Address) IsBYOEAddress() bool {
	return bool(a.Send) && a.Type == AddressTypeExternal
}
