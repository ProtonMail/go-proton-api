package proton_test

import (
	"context"
	"testing"
	"time"

	"github.com/ProtonMail/go-proton-api"
	"github.com/ProtonMail/go-proton-api/server"
	"github.com/stretchr/testify/require"
)

func TestAuth(t *testing.T) {
	s := server.New()
	defer s.Close()

	_, _, err := s.CreateUser("username", "email@pm.me", []byte("password"))
	require.NoError(t, err)

	m := proton.New(
		proton.WithHostURL(s.GetHostURL()),
		proton.WithTransport(proton.InsecureTransport()),
	)
	defer m.Close()

	// Create one session.
	c1, auth1, err := m.NewClientWithLogin(context.Background(), "username", []byte("password"))
	require.NoError(t, err)

	// Revoke all other sessions.
	require.NoError(t, c1.AuthRevokeAll(context.Background()))

	// Create another session.
	c2, _, err := m.NewClientWithLogin(context.Background(), "username", []byte("password"))
	require.NoError(t, err)

	// There should be two sessions.
	sessions, err := c1.AuthSessions(context.Background())
	require.NoError(t, err)
	require.Len(t, sessions, 2)

	// Revoke the first session.
	require.NoError(t, c2.AuthRevoke(context.Background(), auth1.UID))

	// The first session should no longer work.
	require.Error(t, c1.AuthDelete(context.Background()))

	// There should be one session remaining.
	remaining, err := c2.AuthSessions(context.Background())
	require.NoError(t, err)
	require.Len(t, remaining, 1)

	// Delete the last session.
	require.NoError(t, c2.AuthDelete(context.Background()))
}

func TestAuth_Refresh(t *testing.T) {
	s := server.New()
	defer s.Close()

	s.SetAuthLife(10 * time.Second)

	_, _, err := s.CreateUser("username", "email@pm.me", []byte("password"))
	require.NoError(t, err)

	m := proton.New(
		proton.WithHostURL(s.GetHostURL()),
		proton.WithTransport(proton.InsecureTransport()),
	)
	defer m.Close()

	// Create one session.
	c, _, err := m.NewClientWithLogin(context.Background(), "username", []byte("password"))
	require.NoError(t, err)

	// Wait for 5 seconds.
	time.Sleep(5 * time.Second)

	// The client should still be authenticated.
	{
		user, err := c.GetUser(context.Background())
		require.NoError(t, err)
		require.Equal(t, "username", user.Name)
	}

	// Wait for 5 more seconds.
	time.Sleep(5 * time.Second)

	// The client's auth token should have expired, but will be refreshed.
	{
		user, err := c.GetUser(context.Background())
		require.NoError(t, err)
		require.Equal(t, "username", user.Name)
	}
}
