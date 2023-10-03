package backend

import "github.com/google/uuid"

func (b *Backend) CreateCSTicket() string {
	tokenUUID, err := uuid.NewUUID()
	if err != nil {
		return ""
	}

	b.csTicketLock.Lock()
	defer b.csTicketLock.Unlock()

	token := tokenUUID.String()
	b.csTicket = append(b.csTicket, token)
	return token
}
