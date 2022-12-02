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
	"golang.org/x/exp/maps"
)

type Backend struct {
	accounts map[string]*account
	accLock  sync.RWMutex

	attachments map[string]*attachment
	attLock     sync.Mutex

	attData     map[string][]byte
	attDataLock sync.Mutex

	messages map[string]*message
	msgLock  sync.Mutex

	labels  map[string]*label
	lblLock sync.Mutex

	updates     map[ID]update
	updatesLock sync.RWMutex

	srp     map[string]*srp.Server
	srpLock sync.Mutex

	authLife time.Duration
}

func New(authLife time.Duration) *Backend {
	return &Backend{
		accounts:    make(map[string]*account),
		attachments: make(map[string]*attachment),
		attData:     make(map[string][]byte),
		messages:    make(map[string]*message),
		labels:      make(map[string]*label),
		updates:     make(map[ID]update),
		srp:         make(map[string]*srp.Server),

		authLife: authLife,
	}
}

func (b *Backend) SetAuthLife(authLife time.Duration) {
	b.authLife = authLife
}

func (b *Backend) CreateUser(username string, password []byte) (string, error) {
	b.accLock.Lock()
	defer b.accLock.Unlock()

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
}

func (b *Backend) RemoveUser(userID string) error {
	b.accLock.Lock()
	defer b.accLock.Unlock()

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
}

func (b *Backend) RefreshUser(userID string, refresh proton.RefreshFlag) error {
	return b.withAcc(userID, func(acc *account) error {
		updateID, err := b.newUpdate(&userRefreshed{refresh: refresh})
		if err != nil {
			return err
		}

		acc.updateIDs = append(acc.updateIDs, updateID)

		return nil
	})
}

func (b *Backend) CreateUserKey(userID string, password []byte) error {
	b.accLock.Lock()
	defer b.accLock.Unlock()

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
}

func (b *Backend) RemoveUserKey(userID, keyID string) error {
	b.accLock.Lock()
	defer b.accLock.Unlock()

	user, ok := b.accounts[userID]
	if !ok {
		return fmt.Errorf("user %s does not exist", userID)
	}

	idx := xslices.IndexFunc(user.keys, func(key key) bool {
		return key.keyID == keyID
	})

	if idx == -1 {
		return fmt.Errorf("key %s does not exist", keyID)
	}

	user.keys = append(user.keys[:idx], user.keys[idx+1:]...)

	return nil
}

func (b *Backend) CreateAddress(userID, email string, password []byte) (string, error) {
	return withAcc(b, userID, func(acc *account) (string, error) {
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

		addressID := uuid.NewString()

		acc.addresses[addressID] = &address{
			addrID: addressID,
			email:  email,
			order:  len(acc.addresses) + 1,
			keys: []key{{
				keyID: uuid.NewString(),
				key:   armKey,
				tok:   encToken,
				sig:   sigToken,
			}},
		}

		updateID, err := b.newUpdate(&addressCreated{addressID: addressID})
		if err != nil {
			return "", err
		}

		acc.updateIDs = append(acc.updateIDs, updateID)

		return addressID, nil
	})
}

func (b *Backend) CreateAddressKey(userID, addrID string, password []byte) error {
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
}

func (b *Backend) RemoveAddress(userID, addrID string) error {
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
}

func (b *Backend) RemoveAddressKey(userID, addrID, keyID string) error {
	return b.withAcc(userID, func(acc *account) error {
		idx := xslices.IndexFunc(acc.addresses[addrID].keys, func(key key) bool {
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
}

func (b *Backend) CreateMessage(
	userID, addrID string,
	subject string,
	sender *mail.Address,
	toList, ccList, bccList []*mail.Address,
	armBody string,
	mimeType rfc822.MIMEType,
	flags proton.MessageFlag,
	unread, starred bool,
) (string, error) {
	return withAcc(b, userID, func(acc *account) (string, error) {
		return withMessages(b, func(messages map[string]*message) (string, error) {
			msg := newMessage(addrID, subject, sender, toList, ccList, bccList, armBody, mimeType, "")

			msg.flags |= flags
			msg.unread = unread
			msg.starred = starred

			messages[msg.messageID] = msg

			updateID, err := b.newUpdate(&messageCreated{messageID: msg.messageID})
			if err != nil {
				return "", err
			}

			acc.messageIDs = append(acc.messageIDs, msg.messageID)
			acc.updateIDs = append(acc.updateIDs, updateID)

			return msg.messageID, nil
		})
	})
}

func (b *Backend) Encrypt(userID, addrID, decBody string) (string, error) {
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
}

func (b *Backend) withAcc(userID string, fn func(acc *account) error) error {
	b.accLock.RLock()
	defer b.accLock.RUnlock()

	acc, ok := b.accounts[userID]
	if !ok {
		return fmt.Errorf("account %s not found", userID)
	}

	return fn(acc)
}

func (b *Backend) withAccEmail(email string, fn func(acc *account) error) error {
	b.accLock.RLock()
	defer b.accLock.RUnlock()

	for _, acc := range b.accounts {
		for _, addr := range acc.addresses {
			if addr.email == email {
				return fn(acc)
			}
		}
	}

	return fmt.Errorf("account %s not found", email)
}

func withAcc[T any](b *Backend, userID string, fn func(acc *account) (T, error)) (T, error) {
	b.accLock.RLock()
	defer b.accLock.RUnlock()

	for _, acc := range b.accounts {
		if acc.userID == userID {
			return fn(acc)
		}
	}

	return *new(T), fmt.Errorf("account not found")
}

func withAccName[T any](b *Backend, username string, fn func(acc *account) (T, error)) (T, error) {
	b.accLock.RLock()
	defer b.accLock.RUnlock()

	for _, acc := range b.accounts {
		if acc.username == username {
			return fn(acc)
		}
	}

	return *new(T), fmt.Errorf("account not found")
}

func withAccEmail[T any](b *Backend, email string, fn func(acc *account) (T, error)) (T, error) {
	b.accLock.RLock()
	defer b.accLock.RUnlock()

	for _, acc := range b.accounts {
		if _, ok := acc.getAddr(email); ok {
			return fn(acc)
		}
	}

	return *new(T), fmt.Errorf("account not found")
}

func withAccAuth[T any](b *Backend, authUID, authAcc string, fn func(acc *account) (T, error)) (T, error) {
	b.accLock.Lock()
	defer b.accLock.Unlock()

	for _, acc := range b.accounts {
		acc.authLock.Lock()
		defer acc.authLock.Unlock()

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

func (b *Backend) withMessages(fn func(map[string]*message) error) error {
	b.msgLock.Lock()
	defer b.msgLock.Unlock()

	return fn(b.messages)
}

func withMessages[T any](b *Backend, fn func(map[string]*message) (T, error)) (T, error) {
	b.msgLock.Lock()
	defer b.msgLock.Unlock()

	return fn(b.messages)
}

func withAtts[T any](b *Backend, fn func(map[string]*attachment) (T, error)) (T, error) {
	b.attLock.Lock()
	defer b.attLock.Unlock()

	return fn(b.attachments)
}

func (b *Backend) withLabels(fn func(map[string]*label) error) error {
	b.lblLock.Lock()
	defer b.lblLock.Unlock()

	return fn(b.labels)
}

func withLabels[T any](b *Backend, fn func(map[string]*label) (T, error)) (T, error) {
	b.lblLock.Lock()
	defer b.lblLock.Unlock()

	return fn(b.labels)
}

func (b *Backend) newUpdate(event update) (ID, error) {
	return withUpdates(b, func(updates map[ID]update) (ID, error) {
		updateID := ID(len(updates))

		updates[updateID] = event

		return updateID, nil
	})
}

func withUpdates[T any](b *Backend, fn func(map[ID]update) (T, error)) (T, error) {
	b.updatesLock.Lock()
	defer b.updatesLock.Unlock()

	return fn(b.updates)
}
