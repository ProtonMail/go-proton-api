package backend

import (
	"github.com/ProtonMail/go-proton-api"
	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/bradenaw/juniper/xslices"
)

type address struct {
	addrID string
	email  string
	order  int
	status proton.AddressStatus
	keys   []key
}

func (add *address) toAddress() proton.Address {
	return proton.Address{
		ID:    add.addrID,
		Email: add.email,

		Send:    true,
		Receive: true,
		Status:  add.status,

		Order:       add.order,
		DisplayName: add.email,

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
