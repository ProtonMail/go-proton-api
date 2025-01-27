package backend

import (
	"encoding/base64"
	"fmt"

	"github.com/ProtonMail/go-srp"
	"github.com/google/uuid"
	"github.com/henrybear327/go-proton-api"
)

func (b *Backend) NewAuthInfo(username string) (proton.AuthInfo, error) {
	return writeBackendRetErr(b, func(b *unsafeBackend) (proton.AuthInfo, error) {
		return withAccName(b, username, func(acc *account) (proton.AuthInfo, error) {
			server, err := srp.NewServerFromSigned(modulus, acc.verifier, 2048)
			if err != nil {
				log.WithError(err).Errorf("Failed to create SRP Server")
				return proton.AuthInfo{}, fmt.Errorf("failed to create new srp server %w", err)
			}

			challenge, err := server.GenerateChallenge()
			if err != nil {
				log.WithError(err).Errorf("Failed to generate srp challeng")
				return proton.AuthInfo{}, fmt.Errorf("failed to generate srp challend %w", err)
			}

			session := uuid.NewString()

			b.srp[session] = server

			return proton.AuthInfo{
				Version:         4,
				Modulus:         modulus,
				ServerEphemeral: base64.StdEncoding.EncodeToString(challenge),
				Salt:            base64.StdEncoding.EncodeToString(acc.salt),
				SRPSession:      session,
			}, nil
		})
	})
}

func (b *Backend) NewAuth(username string, ephemeral, proof []byte, session string) (proton.Auth, error) {
	return writeBackendRetErr(b, func(b *unsafeBackend) (proton.Auth, error) {
		return withAccName(b, username, func(acc *account) (proton.Auth, error) {
			server, ok := b.srp[session]
			if !ok {
				log.Errorf("Session '%v' not found for user='%v'", session, username)
				return proton.Auth{}, fmt.Errorf("invalid session")
			}

			delete(b.srp, session)

			serverProof, err := server.VerifyProofs(ephemeral, proof)
			if err != nil {
				return proton.Auth{}, fmt.Errorf("invalid proof: %w", err)
			}

			authUID, auth := uuid.NewString(), newAuth(b.authLife)

			acc.auth[authUID] = auth

			return auth.toAuth(acc.userID, authUID, serverProof), nil
		})
	})
}

func (b *Backend) NewAuthRef(authUID, authRef string) (proton.Auth, error) {
	return writeBackendRetErr(b, func(b *unsafeBackend) (proton.Auth, error) {
		for _, acc := range b.accounts {
			auth, ok := acc.auth[authUID]
			if !ok {
				continue
			}

			if auth.ref != authRef {
				return proton.Auth{}, fmt.Errorf("invalid auth ref")
			}

			newAuth := newAuth(b.authLife)

			acc.auth[authUID] = newAuth

			return newAuth.toAuth(acc.userID, authUID, nil), nil
		}

		return proton.Auth{}, fmt.Errorf("invalid auth")
	})
}

func (b *Backend) VerifyAuth(authUID, authAcc string) (string, error) {
	return writeBackendRetErr(b, func(b *unsafeBackend) (string, error) {
		return withAccAuth(b, authUID, authAcc, func(acc *account) (string, error) {
			return acc.userID, nil
		})
	})
}

func (b *Backend) GetSessions(userID string) ([]proton.AuthSession, error) {
	return readBackendRetErr(b, func(b *unsafeBackend) ([]proton.AuthSession, error) {
		return withAcc(b, userID, func(acc *account) ([]proton.AuthSession, error) {
			var sessions []proton.AuthSession

			for authUID, auth := range acc.auth {
				sessions = append(sessions, auth.toAuthSession(authUID))
			}

			return sessions, nil
		})
	})
}

func (b *Backend) DeleteSession(userID, authUID string) error {
	return writeBackendRet(b, func(b *unsafeBackend) error {
		return b.withAcc(userID, func(acc *account) error {
			delete(acc.auth, authUID)

			return nil
		})
	})
}
