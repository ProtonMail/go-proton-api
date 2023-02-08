package proton_test

import (
	"context"
	"errors"
	"net"
	"net/url"
	"testing"

	"github.com/ProtonMail/go-proton-api"
	"github.com/ProtonMail/go-proton-api/server"
	"github.com/stretchr/testify/require"
)

func TestNetError_DropOnWrite(t *testing.T) {
	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	dropListener := proton.NewListener(l, proton.NewDropConn)

	// Use a custom listener that drops all writes.
	dropListener.SetCanWrite(false)

	// Simulate a server that refuses to write.
	s := server.New(server.WithListener(dropListener))
	defer s.Close()

	m := proton.New(proton.WithHostURL(s.GetHostURL()))
	defer m.Close()

	// This should fail with a URL error.
	pingErr := m.Ping(context.Background())

	if urlErr := new(url.Error); !errors.As(pingErr, &urlErr) {
		t.Fatalf("expected a url.Error, got %T: %v", pingErr, pingErr)
	}
}
