package backend

import (
	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/bradenaw/juniper/xslices"
	"github.com/henrybear327/go-proton-api"
)

type address struct {
	addrID      string
	email       string
	displayName string
	order       int
	status      proton.AddressStatus
	addrType    proton.AddressType
	keys        []key
	allowSend   bool
}

func (add *address) toAddress() proton.Address {
	return proton.Address{
		ID:    add.addrID,
		Email: add.email,

		Send:    proton.Bool(add.allowSend),
		Receive: true,
		Status:  add.status,
		Type:    add.addrType,

		Order:       add.order,
		DisplayName: add.displayName,

		Keys: xslices.Map(add.keys, func(key key) proton.Key {
			privKey, err := crypto.NewKeyFromArmored(key.key)
			if err != nil {
				panic(err)
			}

			rawKey, err := privKey.Serialize()
			if err != nil {
				panic(err)
			}

			return proton.Key{
				ID:         key.keyID,
				PrivateKey: rawKey,
				Token:      key.tok,
				Signature:  key.sig,
				Primary:    key == add.keys[0],
				Active:     true,
			}
		}),
	}
}
