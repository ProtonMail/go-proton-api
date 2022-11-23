package proton

import "github.com/ProtonMail/gopenpgp/v2/crypto"

type LinkWalkFunc func([]string, Link, *crypto.KeyRing) error
