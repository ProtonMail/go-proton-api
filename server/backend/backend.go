package backend

import (
	"fmt"
	"net/mail"
	"sync"
	"time"

	"github.com/ProtonMail/gluon/rfc822"
	"github.com/ProtonMail/go-proton-api"
	"github.com/ProtonMail/go-srp"
	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/bradenaw/juniper/xslices"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

var log = logrus.WithField("pkg", "gpa/server/backend")

type Backend struct {
	domain string

	lock        sync.RWMutex
	safeBackend *unsafeBackend
}

// unsafeBackend does not guarantee any thread safety. Methods exposed by Backend should ensure they acquire
// either a Read Lock or Write lock depending on the required changes.
type unsafeBackend struct {
	accounts map[string]*account

	attachments map[string]*attachment

	attData map[string][]byte

	messages map[string]*message

	labels map[string]*label

	updates            map[ID]update
	maxUpdatesPerEvent int

	srp map[string]*srp.Server

	authLife    time.Duration
	enableDedup bool

	csTicket []string
}

func readBackendRet[T any](b *Backend, f func(b *unsafeBackend) T) T {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return f(b.safeBackend)
}

func readBackendRetErr[T any](b *Backend, f func(b *unsafeBackend) (T, error)) (T, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return f(b.safeBackend)
}

func writeBackend(b *Backend, f func(b *unsafeBackend)) {
	b.lock.Lock()
	defer b.lock.Unlock()

	f(b.safeBackend)
}

func writeBackendRet[T any](b *Backend, f func(b *unsafeBackend) T) T {
	b.lock.Lock()
	defer b.lock.Unlock()

	return f(b.safeBackend)
}

func writeBackendRetErr[T any](b *Backend, f func(b *unsafeBackend) (T, error)) (T, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	return f(b.safeBackend)
}

func New(authLife time.Duration, domain string, enableDedup bool) *Backend {
	return &Backend{
		domain: domain,
		safeBackend: &unsafeBackend{
			accounts:           make(map[string]*account),
			attachments:        make(map[string]*attachment),
			attData:            make(map[string][]byte),
			messages:           make(map[string]*message),
			labels:             make(map[string]*label),
			updates:            make(map[ID]update),
			maxUpdatesPerEvent: 0,
			srp:                make(map[string]*srp.Server),
			authLife:           authLife,
			enableDedup:        enableDedup,
		},
	}
}

func (b *Backend) SetAuthLife(authLife time.Duration) {
	writeBackend(b, func(b *unsafeBackend) {
		b.authLife = authLife
	})
}

func (b *Backend) SetMaxUpdatesPerEvent(max int) {
	writeBackend(b, func(b *unsafeBackend) {
		b.maxUpdatesPerEvent = max
	})
}

func (b *Backend) CreateUser(username string, password []byte) (string, error) {
	return writeBackendRetErr(b, func(b *unsafeBackend) (string, error) {
		salt, err := crypto.RandomToken(16)
		if err != nil {
			return "", err
		}

		passphrase, err := hashPassword(password, salt)
		if err != nil {
			return "", err
		}

		srpAuth, err := srp.NewAuthForVerifier(password, modulus, salt)
		if err != nil {
			return "", err
		}

		verifier, err := srpAuth.GenerateVerifier(2048)
		if err != nil {
			return "", err
		}

		armKey, err := GenerateKey(username, username, passphrase, "rsa", 2048)
		if err != nil {
			return "", err
		}

		userID := uuid.NewString()

		b.accounts[userID] = newAccount(userID, username, armKey, salt, verifier)

		return userID, nil
	})
}

func (b *Backend) RemoveUser(userID string) error {
	return writeBackendRet(b, func(b *unsafeBackend) error {
		user, ok := b.accounts[userID]
		if !ok {
			return fmt.Errorf("user %s does not exist", userID)
		}

		for _, labelID := range user.labelIDs {
			delete(b.labels, labelID)
		}

		for _, messageID := range user.messageIDs {
			for _, attID := range b.messages[messageID].attIDs {
				if xslices.CountFunc(maps.Values(b.attachments), func(att *attachment) bool {
					return att.attDataID == b.attachments[attID].attDataID
				}) == 1 {
					delete(b.attData, b.attachments[attID].attDataID)
				}

				delete(b.attachments, attID)
			}

			delete(b.messages, messageID)
		}

		delete(b.accounts, userID)

		return nil
	})
}

func (b *Backend) RefreshUser(userID string, refresh proton.RefreshFlag) error {
	return writeBackendRet(b, func(b *unsafeBackend) error {
		return b.withAcc(userID, func(acc *account) error {
			updateID, err := b.newUpdate(&userRefreshed{refresh: refresh})
			if err != nil {
				return err
			}

			if refresh == proton.RefreshAll {
				acc.updateIDs = []ID{updateID}
			} else {
				acc.updateIDs = append(acc.updateIDs, updateID)
			}

			return nil
		})
	})
}

func (b *Backend) CreateUserKey(userID string, password []byte) error {
	return writeBackendRet(b, func(b *unsafeBackend) error {
		user, ok := b.accounts[userID]
		if !ok {
			return fmt.Errorf("user %s does not exist", userID)
		}

		salt, err := crypto.RandomToken(16)
		if err != nil {
			return err
		}

		passphrase, err := hashPassword(password, salt)
		if err != nil {
			return err
		}

		armKey, err := GenerateKey(user.username, user.username, passphrase, "rsa", 2048)
		if err != nil {
			return err
		}

		user.keys = append(user.keys, key{keyID: uuid.NewString(), key: armKey})

		return nil
	})
}

func (b *Backend) RemoveUserKey(userID, keyID string) error {
	return writeBackendRet(b, func(b *unsafeBackend) error {
		user, ok := b.accounts[userID]
		if !ok {
			return fmt.Errorf("user %s does not exist", userID)
		}

		idx := slices.IndexFunc(user.keys, func(key key) bool {
			return key.keyID == keyID
		})

		if idx == -1 {
			return fmt.Errorf("key %s does not exist", keyID)
		}

		user.keys = append(user.keys[:idx], user.keys[idx+1:]...)

		return nil
	})
}

func (b *Backend) CreateAddress(userID, email string, password []byte, withKey bool, status proton.AddressStatus, addrType proton.AddressType) (string, error) {
	return writeBackendRetErr(b, func(b *unsafeBackend) (string, error) {
		return b.createAddress(userID, email, password, withKey, status, addrType, false, true)
	})
}

func (b *Backend) CreateAddressAsUpdate(userID, email string, password []byte, withKey bool, status proton.AddressStatus, addrType proton.AddressType) (string, error) {
	return writeBackendRetErr(b, func(b *unsafeBackend) (string, error) {
		return b.createAddress(userID, email, password, withKey, status, addrType, true, true)
	})
}

func (b *Backend) CreateAddressWithSendDisabled(
	userID, email string,
	password []byte,
	withKey bool,
	status proton.AddressStatus,
	addrType proton.AddressType,
) (string, error) {
	return writeBackendRetErr(b, func(b *unsafeBackend) (string, error) {
		return b.createAddress(userID, email, password, withKey, status, addrType, false, false)
	})
}

func (b *unsafeBackend) createAddress(
	userID, email string,
	password []byte,
	withKey bool,
	status proton.AddressStatus,
	addrType proton.AddressType,
	issueUpdateInsteadOfCreate bool,
	allowSend bool,
) (string, error) {
	return withAcc(b, userID, func(acc *account) (string, error) {
		var keys []key

		if withKey {
			token, err := crypto.RandomToken(32)
			if err != nil {
				return "", err
			}

			armKey, err := GenerateKey(acc.username, email, token, "rsa", 2048)
			if err != nil {
				return "", err
			}

			passphrase, err := hashPassword([]byte(password), acc.salt)
			if err != nil {
				return "", err
			}

			userKR, err := acc.keys[0].unlock(passphrase)
			if err != nil {
				return "", err
			}

			encToken, sigToken, err := encryptWithSignature(userKR, token)
			if err != nil {
				return "", err
			}

			keys = append(keys, key{
				keyID: uuid.NewString(),
				key:   armKey,
				tok:   encToken,
				sig:   sigToken,
			})
		}

		addressID := uuid.NewString()

		acc.addresses[addressID] = &address{
			addrID:      addressID,
			email:       email,
			displayName: email,
			order:       len(acc.addresses) + 1,
			status:      status,
			addrType:    addrType,
			keys:        keys,
			allowSend:   allowSend,
		}

		var update update
		if issueUpdateInsteadOfCreate {
			update = &addressUpdated{addressID: addressID}
		} else {
			update = &addressCreated{addressID: addressID}
		}

		updateID, err := b.newUpdate(update)
		if err != nil {
			return "", err
		}

		acc.updateIDs = append(acc.updateIDs, updateID)

		return addressID, nil
	})
}

func (b *Backend) ChangeAddressType(userID, addrID string, addrType proton.AddressType) error {
	return writeBackendRet(b, func(b *unsafeBackend) error {
		return b.withAcc(userID, func(acc *account) error {
			for _, addr := range acc.addresses {
				if addr.addrID == addrID {
					addr.addrType = addrType
					return nil
				}
			}
			return fmt.Errorf("no addrID matching %s for user %s", addrID, userID)
		})
	})
}

func (b *Backend) ChangeAddressAllowSend(userID, addrId string, allowSend bool) error {
	return writeBackendRet(b, func(b *unsafeBackend) error {
		return b.withAcc(userID, func(acc *account) error {
			for _, addr := range acc.addresses {
				if addr.addrID == addrId {
					addr.allowSend = allowSend
					return nil
				}
			}
			return fmt.Errorf("no addrID matching %s for user %s", addrId, userID)
		})
	})
}

func (b *Backend) ChangeAddressDisplayName(userID, addrID, displayName string) error {
	return writeBackendRet(b, func(b *unsafeBackend) error {
		return b.withAcc(userID, func(acc *account) error {
			for _, addr := range acc.addresses {
				if addr.addrID == addrID {
					addr.displayName = displayName
					return nil
				}
			}
			return fmt.Errorf("no addrID matching %s for user %s", addrID, userID)
		})
	})
}

func (b *Backend) CreateAddressKey(userID, addrID string, password []byte) error {
	return writeBackendRet(b, func(b *unsafeBackend) error {
		return b.withAcc(userID, func(acc *account) error {
			token, err := crypto.RandomToken(32)
			if err != nil {
				return err
			}

			armKey, err := GenerateKey(acc.username, acc.addresses[addrID].email, token, "rsa", 2048)
			if err != nil {
				return err
			}

			passphrase, err := hashPassword([]byte(password), acc.salt)
			if err != nil {
				return err
			}

			userKR, err := acc.keys[0].unlock(passphrase)
			if err != nil {
				return err
			}

			encToken, sigToken, err := encryptWithSignature(userKR, token)
			if err != nil {
				return err
			}

			acc.addresses[addrID].keys = append(acc.addresses[addrID].keys, key{
				keyID: uuid.NewString(),
				key:   armKey,
				tok:   encToken,
				sig:   sigToken,
			})

			updateID, err := b.newUpdate(&addressUpdated{addressID: addrID})
			if err != nil {
				return err
			}

			acc.updateIDs = append(acc.updateIDs, updateID)

			return nil
		})
	})
}

func (b *Backend) RemoveAddress(userID, addrID string) error {
	return writeBackendRet(b, func(b *unsafeBackend) error {
		return b.withAcc(userID, func(acc *account) error {
			if _, ok := acc.addresses[addrID]; !ok {
				return fmt.Errorf("address %s not found", addrID)
			}

			delete(acc.addresses, addrID)

			updateID, err := b.newUpdate(&addressDeleted{addressID: addrID})
			if err != nil {
				return err
			}

			acc.updateIDs = append(acc.updateIDs, updateID)

			return nil
		})
	})
}

func (b *Backend) RemoveAddressKey(userID, addrID, keyID string) error {
	return writeBackendRet(b, func(b *unsafeBackend) error {
		return b.withAcc(userID, func(acc *account) error {
			idx := slices.IndexFunc(acc.addresses[addrID].keys, func(key key) bool {
				return key.keyID == keyID
			})

			if idx < 0 {
				return fmt.Errorf("key %s not found", keyID)
			}

			acc.addresses[addrID].keys = append(acc.addresses[addrID].keys[:idx], acc.addresses[addrID].keys[idx+1:]...)

			updateID, err := b.newUpdate(&addressUpdated{addressID: addrID})
			if err != nil {
				return err
			}

			acc.updateIDs = append(acc.updateIDs, updateID)

			return nil
		})
	})
}

// TODO: Implement this when we support subscriptions in the test server.
func (b *Backend) CreateSubscription(userID, planID string) error {
	return nil
}

func (b *Backend) CreateMessage(
	userID, addrID string,
	subject string,
	sender *mail.Address,
	toList, ccList, bccList, replytos []*mail.Address,
	armBody string,
	mimeType rfc822.MIMEType,
	flags proton.MessageFlag,
	date time.Time,
	unread, starred bool,
) (string, error) {
	return writeBackendRetErr(b, func(b *unsafeBackend) (string, error) {
		return withAcc(b, userID, func(acc *account) (string, error) {
			return withMessages(b, func(messages map[string]*message) (string, error) {
				msg := newMessage(addrID, subject, sender, toList, ccList, bccList, replytos, armBody, mimeType, "", date)

				msg.flags |= flags
				msg.unread = unread
				msg.starred = starred

				addrListEqual := func(l1 []*mail.Address, l2 []*mail.Address) bool {
					s1 := xslices.Map(l1, func(addr *mail.Address) string {
						return addr.Address
					})
					s2 := xslices.Map(l2, func(addr *mail.Address) string {
						return addr.Address
					})

					return slices.Equal(s1, s2)
				}

				var foundDuplicate bool

				if b.enableDedup {
					for _, m := range messages {
						if m.addrID != msg.addrID {
							continue
						}

						toEqual := addrListEqual(m.toList, msg.toList)
						bccEqual := addrListEqual(m.bccList, msg.bccList)
						ccEqual := addrListEqual(m.ccList, msg.ccList)

						if m.sender.Address == msg.sender.Address &&
							toEqual &&
							bccEqual &&
							ccEqual &&
							m.subject == msg.subject {
							msg.messageID = m.messageID
							foundDuplicate = true
							break
						}
					}
				}

				if !foundDuplicate {
					messages[msg.messageID] = msg

					updateID, err := b.newUpdate(&messageCreated{messageID: msg.messageID})
					if err != nil {
						return "", err
					}

					acc.messageIDs = append(acc.messageIDs, msg.messageID)
					acc.updateIDs = append(acc.updateIDs, updateID)
				}

				return msg.messageID, nil
			})
		})
	})
}

func (b *Backend) Encrypt(userID, addrID, decBody string) (string, error) {
	return readBackendRetErr(b, func(b *unsafeBackend) (string, error) {
		return withAcc(b, userID, func(acc *account) (string, error) {
			pubKey, err := acc.addresses[addrID].keys[0].getPubKey()
			if err != nil {
				return "", err
			}

			kr, err := crypto.NewKeyRing(pubKey)
			if err != nil {
				return "", err
			}

			enc, err := kr.Encrypt(crypto.NewPlainMessageFromString(decBody), nil)
			if err != nil {
				return "", err
			}

			return enc.GetArmored()
		})
	})
}

func (b *unsafeBackend) withAcc(userID string, fn func(acc *account) error) error {
	acc, ok := b.accounts[userID]
	if !ok {
		return fmt.Errorf("account %s not found", userID)
	}

	return fn(acc)
}

func (b *unsafeBackend) withAccEmail(email string, fn func(acc *account) error) error {
	for _, acc := range b.accounts {
		for _, addr := range acc.addresses {
			if addr.email == email {
				return fn(acc)
			}
		}
	}

	return fmt.Errorf("account %s not found", email)
}

func withAcc[T any](b *unsafeBackend, userID string, fn func(acc *account) (T, error)) (T, error) {
	for _, acc := range b.accounts {
		if acc.userID == userID {
			return fn(acc)
		}
	}

	return *new(T), fmt.Errorf("account not found")
}

func withAccName[T any](b *unsafeBackend, username string, fn func(acc *account) (T, error)) (T, error) {
	for _, acc := range b.accounts {
		if acc.username == username {
			return fn(acc)
		}
	}

	return *new(T), fmt.Errorf("account not found")
}

func withAccEmail[T any](b *unsafeBackend, email string, fn func(acc *account) (T, error)) (T, error) {
	for _, acc := range b.accounts {
		if _, ok := acc.getAddr(email); ok {
			return fn(acc)
		}
	}

	return *new(T), fmt.Errorf("account not found")
}

func withAccAuth[T any](b *unsafeBackend, authUID, authAcc string, fn func(acc *account) (T, error)) (T, error) {
	for _, acc := range b.accounts {
		val, ok := acc.auth[authUID]
		if !ok {
			continue
		}

		if time.Since(val.creation) > b.authLife {
			acc.auth[authUID] = auth{ref: val.ref, creation: val.creation}
		} else if val.acc == authAcc {
			return fn(acc)
		}
	}

	return *new(T), fmt.Errorf("account not found")
}

func (b *unsafeBackend) withMessages(fn func(map[string]*message) error) error {
	return fn(b.messages)
}

func withMessages[T any](b *unsafeBackend, fn func(map[string]*message) (T, error)) (T, error) {
	return fn(b.messages)
}

func withAtts[T any](b *unsafeBackend, fn func(map[string]*attachment) (T, error)) (T, error) {
	return fn(b.attachments)
}

func (b *unsafeBackend) withLabels(fn func(map[string]*label) error) error {
	return fn(b.labels)
}

func withLabels[T any](b *unsafeBackend, fn func(map[string]*label) (T, error)) (T, error) {
	return fn(b.labels)
}

func (b *unsafeBackend) newUpdate(event update) (ID, error) {
	return withUpdates(b, func(updates map[ID]update) (ID, error) {
		updateID := ID(len(updates))

		updates[updateID] = event

		return updateID, nil
	})
}

func withUpdates[T any](b *unsafeBackend, fn func(map[ID]update) (T, error)) (T, error) {
	return fn(b.updates)
}
