package proton_test

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ProtonMail/go-proton-api"
	"github.com/ProtonMail/go-proton-api/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectionReuse(t *testing.T) {
	s := server.New()
	defer s.Close()

	netCtl := proton.NewNetCtl()

	var dialed int

	netCtl.OnDial(func(net.Conn) {
		dialed++
	})

	m := proton.New(
		proton.WithHostURL(s.GetHostURL()),
		proton.WithTransport(proton.NewDialer(netCtl, &tls.Config{InsecureSkipVerify: true}).GetRoundTripper()),
	)

	// This should succeed; the resulting connection should be reused.
	require.NoError(t, m.Ping(context.Background()))

	// We should have dialed once.
	require.Equal(t, 1, dialed)

	// This should succeed; we should not re-dial.
	require.NoError(t, m.Ping(context.Background()))

	// We should not have re-dialed.
	require.Equal(t, 1, dialed)
}

func TestAuthRefresh(t *testing.T) {
	s := server.New()
	defer s.Close()

	_, _, err := s.CreateUser("user", "email@pm.me", []byte("pass"))
	require.NoError(t, err)

	m := proton.New(
		proton.WithHostURL(s.GetHostURL()),
		proton.WithTransport(proton.InsecureTransport()),
	)

	c1, auth, err := m.NewClientWithLogin(context.Background(), "user", []byte("pass"))
	require.NoError(t, err)
	defer c1.Close()

	c2, auth, err := m.NewClientWithRefresh(context.Background(), auth.UID, auth.RefreshToken)
	require.NoError(t, err)
	defer c2.Close()
}

func TestHandleTooManyRequests(t *testing.T) {
	var numCalls int

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		numCalls++

		if numCalls < 5 {
			w.WriteHeader(http.StatusTooManyRequests)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()

	m := proton.New(
		proton.WithHostURL(ts.URL),
		proton.WithRetryCount(5),
	)

	// The call should succeed because the 5th retry should succeed (429s are retried).
	c := m.NewClient("", "", "", time.Now().Add(time.Hour))
	defer c.Close()

	if _, err := c.GetAddresses(context.Background()); err != nil {
		t.Fatal("got unexpected error", err)
	}

	// The server should be called 5 times.
	// The first four calls should return 429 and the last call should return 200.
	if numCalls != 5 {
		t.Fatal("expected numCalls to be 5, instead got", numCalls)
	}
}

func TestHandleTooManyRequestsRetryAfter(t *testing.T) {
	getDelay := func(iCal int) time.Duration {
		return time.Duration(5*1<<iCal) * time.Second
	}

	iRetry := -1
	lastCall := time.Now()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		currentCall := time.Now()

		if iRetry >= 0 {
			delay := getDelay(iRetry)
			assert.False(t, currentCall.Before(
				lastCall.Add(delay)),
				"Delay was %v but expected to have %v",
				currentCall.Sub(lastCall),
				delay,
			)
		}

		iRetry++
		lastCall = currentCall

		// test defaul 10sec
		if iRetry == 1 {
			w.Header().Set("Retry-After", "something")
		} else {
			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", getDelay(iRetry).Seconds()))
		}
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer ts.Close()

	m := proton.New(
		proton.WithHostURL(ts.URL),
		proton.WithRetryCount(3),
	)

	c := m.NewClient("", "", "", time.Now().Add(time.Hour))
	defer c.Close()

	_, err := c.GetAddresses(context.Background())
	require.Error(t, err)
}

func TestHandleUnprocessableEntity(t *testing.T) {
	var numCalls int

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		numCalls++
		w.WriteHeader(http.StatusUnprocessableEntity)
	}))
	defer ts.Close()

	m := proton.New(
		proton.WithHostURL(ts.URL),
		proton.WithRetryCount(5),
	)

	// The call should fail because the first call should fail (422s are not retried).
	c := m.NewClient("", "", "", time.Now().Add(time.Hour))
	defer c.Close()

	if _, err := c.GetAddresses(context.Background()); err == nil {
		t.Fatal("expected error, instead got", err)
	}

	// The server should be called 1 time.
	// The first call should return 422.
	if numCalls != 1 {
		t.Fatal("expected numCalls to be 1, instead got", numCalls)
	}
}

func TestHandleDialFailure(t *testing.T) {
	var numCalls int

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		numCalls++
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	m := proton.New(
		proton.WithHostURL(ts.URL),
		proton.WithRetryCount(5),
		proton.WithTransport(newFailingRoundTripper(5)),
	)

	// The call should succeed because the last retry should succeed (dial errors are retried).
	c := m.NewClient("", "", "", time.Now().Add(time.Hour))
	defer c.Close()

	if _, err := c.GetAddresses(context.Background()); err != nil {
		t.Fatal("got unexpected error", err)
	}

	// The server should be called 1 time.
	// The first 4 attempts don't reach the server.
	if numCalls != 1 {
		t.Fatal("expected numCalls to be 1, instead got", numCalls)
	}
}

func TestHandleTooManyDialFailures(t *testing.T) {
	var numCalls int

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		numCalls++
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// The failingRoundTripper will fail the first 10 times it is used.
	// This is more than the number of retries we permit.
	// Thus, dials will fail.
	m := proton.New(
		proton.WithHostURL(ts.URL),
		proton.WithRetryCount(5),
		proton.WithTransport(newFailingRoundTripper(10)),
	)

	// The call should fail because every dial will fail and we'll run out of retries.
	c := m.NewClient("", "", "", time.Now().Add(time.Hour))
	defer c.Close()

	if _, err := c.GetAddresses(context.Background()); err == nil {
		t.Fatal("expected error, instead got", err)
	}

	// The server should never be called.
	if numCalls != 0 {
		t.Fatal("expected numCalls to be 0, instead got", numCalls)
	}
}

func TestRetriesWithContextTimeout(t *testing.T) {
	var numCalls int

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		numCalls++

		if numCalls < 5 {
			w.WriteHeader(http.StatusTooManyRequests)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		time.Sleep(time.Second)
	}))
	defer ts.Close()

	m := proton.New(
		proton.WithHostURL(ts.URL),
		proton.WithRetryCount(5),
	)

	// Timeout after 1s.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Theoretically, this should succeed; on the fifth retry, we'll get StatusOK.
	// However, that will take at least >5s, and we only allow 1s in the context.
	// Thus, it will fail.
	c := m.NewClient("", "", "", time.Now().Add(time.Hour))
	defer c.Close()

	if _, err := c.GetAddresses(ctx); err == nil {
		t.Fatal("expected error, instead got", err)
	}
}

func TestReturnErrNoConnection(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// We will fail more times than we retry, so requests should fail with ErrNoConnection.
	m := proton.New(
		proton.WithHostURL(ts.URL),
		proton.WithRetryCount(5),
		proton.WithTransport(newFailingRoundTripper(10)),
	)

	// The call should fail because every dial will fail and we'll run out of retries.
	c := m.NewClient("", "", "", time.Now().Add(time.Hour))
	defer c.Close()

	if _, err := c.GetAddresses(context.Background()); err == nil {
		t.Fatal("expected error, instead got", err)
	}
}

func TestStatusCallbacks(t *testing.T) {
	s := server.New()
	defer s.Close()

	ctl := proton.NewNetCtl()

	m := proton.New(
		proton.WithHostURL(s.GetHostURL()),
		proton.WithTransport(proton.NewDialer(ctl, &tls.Config{InsecureSkipVerify: true}).GetRoundTripper()),
	)

	statusCh := make(chan proton.Status, 1)

	m.AddStatusObserver(func(status proton.Status) {
		statusCh <- status
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctl.Disable()

	require.Error(t, m.Ping(ctx))
	require.Equal(t, proton.StatusDown, <-statusCh)

	ctl.Enable()

	require.NoError(t, m.Ping(ctx))
	require.Equal(t, proton.StatusUp, <-statusCh)

	ctl.SetReadLimit(1)

	require.Error(t, m.Ping(ctx))
	require.Equal(t, proton.StatusDown, <-statusCh)

	ctl.SetReadLimit(0)

	require.NoError(t, m.Ping(ctx))
	require.Equal(t, proton.StatusUp, <-statusCh)
}

type failingRoundTripper struct {
	http.RoundTripper

	fails, calls int
}

func newFailingRoundTripper(fails int) http.RoundTripper {
	return &failingRoundTripper{
		RoundTripper: http.DefaultTransport,
		fails:        fails,
	}
}

func (rt *failingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.calls++

	if rt.calls < rt.fails {
		return nil, errors.New("simulating network error")
	}

	return rt.RoundTripper.RoundTrip(req)
}
