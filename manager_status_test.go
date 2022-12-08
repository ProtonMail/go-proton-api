package proton_test

import (
	"context"
	"crypto/tls"
	"net"
	"testing"

	"github.com/ProtonMail/go-proton-api"
	"github.com/ProtonMail/go-proton-api/server"
	"github.com/stretchr/testify/require"
)

func TestStatus(t *testing.T) {
	s := server.New()
	defer s.Close()

	netCtl := proton.NewNetCtl()

	m := proton.New(
		proton.WithHostURL(s.GetHostURL()),
		proton.WithTransport(proton.NewDialer(netCtl, &tls.Config{InsecureSkipVerify: true}).GetRoundTripper()),
	)

	var (
		called int
		status proton.Status
	)

	m.AddStatusObserver(func(val proton.Status) {
		called++
		status = val
	})

	// This should succeed.
	require.NoError(t, m.Ping(context.Background()))

	// Status should not have been called yet.
	require.Zero(t, called)

	// Now we simulate a network failure.
	netCtl.Disable()

	// This should fail.
	require.Error(t, m.Ping(context.Background()))

	// Status should have been called once and status should indicate network is down.
	require.Equal(t, 1, called)
	require.Equal(t, proton.StatusDown, status)

	// Now we simulate a network restoration.
	netCtl.Enable()

	// This should succeed.
	require.NoError(t, m.Ping(context.Background()))

	// Status should have been called twice and status should indicate network is up.
	require.Equal(t, 2, called)
	require.Equal(t, proton.StatusUp, status)
}

func TestStatus_NoDial(t *testing.T) {
	s := server.New()
	defer s.Close()

	netCtl := proton.NewNetCtl()

	m := proton.New(
		proton.WithHostURL(s.GetHostURL()),
		proton.WithTransport(proton.NewDialer(netCtl, &tls.Config{InsecureSkipVerify: true}).GetRoundTripper()),
	)

	var (
		called int
		status proton.Status
	)

	m.AddStatusObserver(func(val proton.Status) {
		called++
		status = val
	})

	// Disable dialing.
	netCtl.SetCanDial(false)

	// This should fail.
	require.Error(t, m.Ping(context.Background()))

	// Status should have been called once and status should indicate network is down.
	require.Equal(t, 1, called)
	require.Equal(t, proton.StatusDown, status)
}

func TestStatus_NoRead(t *testing.T) {
	s := server.New()
	defer s.Close()

	netCtl := proton.NewNetCtl()

	m := proton.New(
		proton.WithHostURL(s.GetHostURL()),
		proton.WithTransport(proton.NewDialer(netCtl, &tls.Config{InsecureSkipVerify: true}).GetRoundTripper()),
	)

	var (
		called int
		status proton.Status
	)

	m.AddStatusObserver(func(val proton.Status) {
		called++
		status = val
	})

	// Disable reading.
	netCtl.SetCanRead(false)

	// This should fail.
	require.Error(t, m.Ping(context.Background()))

	// Status should have been called once and status should indicate network is down.
	require.Equal(t, 1, called)
	require.Equal(t, proton.StatusDown, status)
}

func TestStatus_NoWrite(t *testing.T) {
	s := server.New()
	defer s.Close()

	netCtl := proton.NewNetCtl()

	m := proton.New(
		proton.WithHostURL(s.GetHostURL()),
		proton.WithTransport(proton.NewDialer(netCtl, &tls.Config{InsecureSkipVerify: true}).GetRoundTripper()),
	)

	var (
		called int
		status proton.Status
	)

	m.AddStatusObserver(func(val proton.Status) {
		called++
		status = val
	})

	// Disable writing.
	netCtl.SetCanWrite(false)

	// This should fail.
	require.Error(t, m.Ping(context.Background()))

	// Status should have been called once and status should indicate network is down.
	require.Equal(t, 1, called)
	require.Equal(t, proton.StatusDown, status)
}

func TestStatus_NoReadExistingConn(t *testing.T) {
	s := server.New()
	defer s.Close()

	_, _, err := s.CreateUser("user", []byte("pass"))
	require.NoError(t, err)

	netCtl := proton.NewNetCtl()

	var dialed int

	netCtl.OnDial(func(net.Conn) {
		dialed++
	})

	m := proton.New(
		proton.WithHostURL(s.GetHostURL()),
		proton.WithTransport(proton.NewDialer(netCtl, &tls.Config{InsecureSkipVerify: true}).GetRoundTripper()),
	)

	// This should succeed.
	c, _, err := m.NewClientWithLogin(context.Background(), "user", []byte("pass"))
	require.NoError(t, err)
	defer c.Close()

	// We should have dialed once.
	require.Equal(t, 1, dialed)

	// Disable reading on the existing connection.
	netCtl.SetCanRead(false)

	// This should fail because we won't be able to read the response.
	require.Error(t, getErr(c.GetUser(context.Background())))

	// We should still have dialed once; the connection should have been reused.
	require.Equal(t, 1, dialed)
}

func TestStatus_NoWriteExistingConn(t *testing.T) {
	s := server.New()
	defer s.Close()

	_, _, err := s.CreateUser("user", []byte("pass"))
	require.NoError(t, err)

	netCtl := proton.NewNetCtl()

	var dialed int

	netCtl.OnDial(func(net.Conn) {
		dialed++
	})

	m := proton.New(
		proton.WithHostURL(s.GetHostURL()),
		proton.WithTransport(proton.NewDialer(netCtl, &tls.Config{InsecureSkipVerify: true}).GetRoundTripper()),
		proton.WithRetryCount(0),
	)

	// This should succeed.
	c, _, err := m.NewClientWithLogin(context.Background(), "user", []byte("pass"))
	require.NoError(t, err)
	defer c.Close()

	// We should have dialed once.
	require.Equal(t, 1, dialed)

	// Disable reading on the existing connection.
	netCtl.SetCanWrite(false)

	// This should fail because we won't be able to write the request.
	require.Error(t, c.LabelMessages(context.Background(), []string{"messageID"}, proton.TrashLabel))

	// We should still have dialed twice; the connection could not be reused because the write failed.
	require.Equal(t, 2, dialed)
}

func TestStatus_ContextCancel(t *testing.T) {
	s := server.New()
	defer s.Close()

	m := proton.New(proton.WithHostURL(s.GetHostURL()))

	var called int

	m.AddStatusObserver(func(val proton.Status) {
		called++
	})

	// Create a context that will be canceled.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// This should fail because the context is canceled.
	require.Error(t, m.Ping(ctx))

	// Status should not have been called; this was not a network error.
	require.Zero(t, called)
}

func TestStatus_ContextTimeout(t *testing.T) {
	s := server.New()
	defer s.Close()

	m := proton.New(proton.WithHostURL(s.GetHostURL()))

	var called int

	m.AddStatusObserver(func(val proton.Status) {
		called++
	})

	// Create a context that will time out.
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	cancel()

	// This should fail because the context is canceled.
	require.Error(t, m.Ping(ctx))

	// Status should have been called; this was a network error (took too long).
	require.NotZero(t, called)
}

func getErr[T any](_ T, err error) error {
	return err
}
