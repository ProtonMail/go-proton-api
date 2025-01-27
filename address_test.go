package proton_test

import (
	"context"
	"testing"

	"github.com/henrybear327/go-proton-api"
	"github.com/henrybear327/go-proton-api/server"
	"github.com/stretchr/testify/require"
)

func TestAddress_DisplayName(t *testing.T) {
	s := server.New()
	defer s.Close()

	// Create a user on the server.
	userID, _, err := s.CreateUser("user", []byte("pass"))
	require.NoError(t, err)
	addr1, err := s.CreateAddress(userID, "user1@pm.me", []byte("pass"))
	require.NoError(t, err)
	require.NoError(t, s.ChangeAddressDisplayName(userID, addr1, "User 1"))

	addr2, err := s.CreateAddress(userID, "user2@pm.me", []byte("pass"))
	require.NoError(t, err)
	require.NoError(t, s.ChangeAddressDisplayName(userID, addr2, "User 2"))

	require.NoError(t, err)

	m := proton.New(
		proton.WithHostURL(s.GetHostURL()),
		proton.WithTransport(proton.InsecureTransport()),
	)
	defer m.Close()

	// Create one session for the user.
	c, auth, err := m.NewClientWithLogin(context.Background(), "user", []byte("pass"))
	require.NoError(t, err)
	require.Equal(t, userID, auth.UserID)

	// Get addresses for the user.
	addrs, err := c.GetAddresses(context.Background())
	require.NoError(t, err)

	for _, addr := range addrs {
		switch addr.ID {
		case addr1:
			require.Equal(t, addr.Email, "user1@pm.me")
			require.Equal(t, addr.DisplayName, "User 1")
		case addr2:
			require.Equal(t, addr.Email, "user2@pm.me")
			require.Equal(t, addr.DisplayName, "User 2")
		}
	}
}

func TestAddress_Types(t *testing.T) {
	s := server.New()
	defer s.Close()

	// Create a user on the server.
	userID, _, err := s.CreateUser("user", []byte("pass"))
	require.NoError(t, err)
	id2, err := s.CreateAddress(userID, "user@alias.com", []byte("pass"))
	require.NoError(t, err)
	require.NoError(t, s.ChangeAddressType(userID, id2, proton.AddressTypeAlias))
	id3, err := s.CreateAddress(userID, "user@custom.com", []byte("pass"))
	require.NoError(t, err)
	require.NoError(t, s.ChangeAddressType(userID, id3, proton.AddressTypeCustom))
	id4, err := s.CreateAddress(userID, "user@premium.com", []byte("pass"))
	require.NoError(t, err)
	require.NoError(t, s.ChangeAddressType(userID, id4, proton.AddressTypePremium))
	id5, err := s.CreateAddress(userID, "user@external.com", []byte("pass"))
	require.NoError(t, err)
	require.NoError(t, s.ChangeAddressType(userID, id5, proton.AddressTypeExternal))

	m := proton.New(
		proton.WithHostURL(s.GetHostURL()),
		proton.WithTransport(proton.InsecureTransport()),
	)
	defer m.Close()

	// Create one session for the user.
	c, auth, err := m.NewClientWithLogin(context.Background(), "user", []byte("pass"))
	require.NoError(t, err)
	require.Equal(t, userID, auth.UserID)

	// Get addresses for the user.
	addrs, err := c.GetAddresses(context.Background())
	require.NoError(t, err)

	for _, addr := range addrs {
		switch addr.ID {
		case id2:
			require.Equal(t, addr.Email, "user@alias.com")
			require.Equal(t, addr.Type, proton.AddressTypeAlias)
		case id3:
			require.Equal(t, addr.Email, "user@custom.com")
			require.Equal(t, addr.Type, proton.AddressTypeCustom)
		case id4:
			require.Equal(t, addr.Email, "user@premium.com")
			require.Equal(t, addr.Type, proton.AddressTypePremium)
		case id5:
			require.Equal(t, addr.Email, "user@external.com")
			require.Equal(t, addr.Type, proton.AddressTypeExternal)
		default:
			require.Equal(t, addr.Email, "user@proton.local")
			require.Equal(t, addr.Type, proton.AddressTypeOriginal)
		}
	}

}
