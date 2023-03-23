package proton

import (
	"fmt"
	"runtime"

	"github.com/ProtonMail/gluon/queue"
	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/bradenaw/juniper/parallel"
)

func Unlock(user User, addresses []Address, saltedKeyPass []byte, panicHandler queue.PanicHandler) (*crypto.KeyRing, map[string]*crypto.KeyRing, error) {
	userKR, err := user.Keys.Unlock(saltedKeyPass, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unlock user keys: %w", err)
	} else if userKR.CountDecryptionEntities() == 0 {
		return nil, nil, fmt.Errorf("failed to unlock any user keys")
	}

	addrKRs := make(map[string]*crypto.KeyRing)

	for idx, addrKR := range parallel.Map(runtime.NumCPU(), addresses, func(addr Address) *crypto.KeyRing {
		defer func() {
			if panicHandler != nil {
				panicHandler.HandlePanic()
			}
		}()

		return addr.Keys.TryUnlock(saltedKeyPass, userKR)
	}) {
		if addrKR == nil {
			continue
		} else if addrKR.CountDecryptionEntities() == 0 {
			continue
		}

		addrKRs[addresses[idx].ID] = addrKR
	}

	if len(addrKRs) == 0 {
		return nil, nil, fmt.Errorf("failed to unlock any address keys")
	}

	return userKR, addrKRs, nil
}
