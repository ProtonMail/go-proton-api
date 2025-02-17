package proton

import "fmt"

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

func (s AddressStatus) String() string {
	statusStrings := [...]string{"disabled", "enabled", "deleting" }
	 if s < AddressStatusDisabled || s > AddressStatusDeleting {
		return fmt.Sprintf("Unknown Status (%d)", s)
	}
	return statusStrings[s]
}

type AddressType int

const (
	AddressTypeOriginal AddressType = iota + 1
	AddressTypeAlias
	AddressTypeCustom
	AddressTypePremium
	AddressTypeExternal
)

func (a AddressType) String() string {
	typeStrings := [...]string{"original", "alias", "custom", "premium", "external" }
	 if a < AddressTypeOriginal || a > AddressTypeExternal {
		return fmt.Sprintf("Unknown Status (%d)", a)
	}

	// Proton API defines the start type as `iota + 1`
	return typeStrings[a-1]
}
