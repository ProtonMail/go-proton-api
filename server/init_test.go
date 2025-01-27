package server

import "github.com/henrybear327/go-proton-api/server/backend"

func init() {
	backend.GenerateKey = backend.FastGenerateKey
}
