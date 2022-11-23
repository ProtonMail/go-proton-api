package proton

import (
	"fmt"
	"runtime"

	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/bradenaw/juniper/parallel"
)

func Unlock(user User, addresses []Address, saltedKeyPass []byte) (*crypto.KeyRing, map[string]*crypto.KeyRing, error) {
	userKR, err := user.Keys.Unlock(saltedKeyPass, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unlock user keys: %w", err)
	}

	addrKRs := make(map[string]*crypto.KeyRing)

	for idx, addrKR := range parallel.Map(runtime.NumCPU(), addresses, func(addr Address) *crypto.KeyRing {
		return addr.Keys.TryUnlock(saltedKeyPass, userKR)
	}) {
		if addrKR != nil {
			addrKRs[addresses[idx].ID] = addrKR
		}
	}

	if len(addrKRs) == 0 {
		return nil, nil, fmt.Errorf("failed to unlock any address keys")
	}

	return userKR, addrKRs, nil
}
