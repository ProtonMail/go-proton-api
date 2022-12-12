package server

import "github.com/ProtonMail/go-proton-api/server/backend"

func init() {
	backend.GenerateKey = backend.FastGenerateKey
}
