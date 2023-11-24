package backend

import "github.com/google/uuid"

func (b *Backend) CreateCSTicket() string {
	tokenUUID, err := uuid.NewUUID()
	if err != nil {
		return ""
	}

	return writeBackendRet(b, func(b *unsafeBackend) string {
		token := tokenUUID.String()
		b.csTicket = append(b.csTicket, token)
		return token
	})
}

func (b *Backend) GetCSTicket(token string) bool {
	return readBackendRet(b, func(b *unsafeBackend) bool {
		for _, ticket := range b.csTicket {
			if ticket == token {
				return true
			}
		}
		return false
	})
}
