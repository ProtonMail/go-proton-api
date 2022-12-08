package backend

import (
	"flag"
	"fmt"

	"github.com/ProtonMail/go-proton-api"
)

func (s *Backend) RunQuarkCommand(command string, args ...string) (any, error) {
	switch command {
	case "encryption:id":
		return s.quarkEncryptionID(args...)

	case "user:create":
		return s.quarkUserCreate(args...)

	case "user:create:address":
		return s.quarkUserCreateAddress(args...)

	default:
		return nil, fmt.Errorf("unknown command: %s", command)
	}
}

func (s *Backend) quarkEncryptionID(args ...string) (string, error) {
	fs := flag.NewFlagSet("encryption:id", flag.ContinueOnError)

	// Required arguments.
	// arg0: value

	decrypt := fs.Bool("decrypt", false, "decrypt the given encrypted ID")

	if err := fs.Parse(args); err != nil {
		return "", err
	}

	// TODO: Encrypt/decrypt are currently no-op.
	if *decrypt {
		return fs.Arg(0), nil
	} else {
		return fs.Arg(0), nil
	}
}

func (s *Backend) quarkUserCreate(args ...string) (proton.User, error) {
	fs := flag.NewFlagSet("user:create", flag.ContinueOnError)

	// Required arguments.
	name := fs.String("name", "", "new user's name")
	pass := fs.String("password", "", "new user's password")

	// Optional arguments.
	newAddr := fs.Bool("create-address", false, "create the user's default address, will not automatically setup the address key")
	genKeys := fs.String("gen-keys", "", "generate new address keys for the user")

	if err := fs.Parse(args); err != nil {
		return proton.User{}, err
	}

	userID, err := s.CreateUser(*name, []byte(*pass))
	if err != nil {
		return proton.User{}, fmt.Errorf("failed to create user: %w", err)
	}

	// TODO: Create keys of different types (we always use RSA2048).
	if *newAddr || *genKeys != "" {
		if _, err := s.CreateAddress(userID, *name+"@"+s.domain, []byte(*pass), *genKeys != ""); err != nil {
			return proton.User{}, fmt.Errorf("failed to create address with keys: %w", err)
		}
	}

	return s.GetUser(userID)
}

func (s *Backend) quarkUserCreateAddress(args ...string) (proton.Address, error) {
	fs := flag.NewFlagSet("user:create:address", flag.ContinueOnError)

	// Required arguments.
	// arg0: userID
	// arg1: password
	// arg2: email

	// Optional arguments.
	genKeys := fs.String("gen-keys", "", "generate new address keys for the user")

	if err := fs.Parse(args); err != nil {
		return proton.Address{}, err
	}

	// TODO: Create keys of different types (we always use RSA2048).
	addrID, err := s.CreateAddress(fs.Arg(0), fs.Arg(2), []byte(fs.Arg(1)), *genKeys != "")
	if err != nil {
		return proton.Address{}, fmt.Errorf("failed to create address with keys: %w", err)
	}

	return s.GetAddress(fs.Arg(0), addrID)
}
