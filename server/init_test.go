package server

import (
	"github.com/ProtonMail/go-proton-api/server/backend"
	"github.com/ProtonMail/gopenpgp/v2/crypto"
)

func init() {
	key, err := crypto.GenerateKey("name", "email", "rsa", 1024)
	if err != nil {
		panic(err)
	}

	backend.GenerateKey = func(_, _ string, passphrase []byte, _ string, _ int) (string, error) {
		encKey, err := key.Lock(passphrase)
		if err != nil {
			return "", err
		}

		return encKey.Armor()
	}
}
