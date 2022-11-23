package proton_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ProtonMail/go-proton-api"
	"github.com/ProtonMail/go-proton-api/server"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestAutomaticAuthRefresh(t *testing.T) {
	wantAuth := proton.Auth{
		UID:          "testUID",
		AccessToken:  "testAcc",
		RefreshToken: "testRef",
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/core/v4/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(wantAuth); err != nil {
			panic(err)
		}
	})

	mux.HandleFunc("/core/v4/users", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	var gotAuth proton.Auth

	// Create a new client.
	c := proton.New(proton.WithHostURL(ts.URL)).NewClient("uid", "acc", "ref", time.Now().Add(-time.Second))
	defer c.Close()

	// Register an auth handler.
	c.AddAuthHandler(func(auth proton.Auth) { gotAuth = auth })

	// Make a request with an access token that already expired one second ago.
	if _, err := c.GetUser(context.Background()); err != nil {
		t.Fatal("got unexpected error", err)
	}

	// The auth callback should have been called.
	if !cmp.Equal(gotAuth, wantAuth) {
		t.Fatal("got unexpected auth", gotAuth)
	}
}

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
