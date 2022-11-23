package proton_test

import (
	"bytes"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ProtonMail/go-proton-api"
)

func TestNetCtl_ReadLimit(t *testing.T) {
	// Create a test http server that writes 100 bytes.
	// Including the header, this is 217 bytes (100 bytes + 117 bytes).
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(make([]byte, 100)); err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	// Create a new net controller.
	netCtl := proton.NewNetCtl()

	// Create a new http client with the dialer.
	client := &http.Client{
		Transport: proton.NewDialer(netCtl, &tls.Config{InsecureSkipVerify: true}).GetRoundTripper(),
	}

	// Set the read limit to 300 bytes -- the first request should succeed, the second should fail.
	netCtl.SetReadLimit(300)

	// This should succeed.
	if resp, err := client.Get(ts.URL); err != nil {
		t.Fatal(err)
	} else {
		resp.Body.Close()
	}

	// This should fail.
	if _, err := client.Get(ts.URL); err == nil {
		t.Fatal("expected error")
	}
}

func TestNetCtl_WriteLimit(t *testing.T) {
	// Create a test http server that reads the given body.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	// Create a new net controller.
	netCtl := proton.NewNetCtl()

	// Create a new http client with the dialer.
	client := &http.Client{
		Transport: proton.NewDialer(netCtl, &tls.Config{InsecureSkipVerify: true}).GetRoundTripper(),
	}

	// Set the read limit to 300 bytes -- the first request should succeed, the second should fail.
	netCtl.SetWriteLimit(300)

	// This should succeed.
	if resp, err := client.Post(ts.URL, "application/octet-stream", bytes.NewReader(make([]byte, 100))); err != nil {
		t.Fatal(err)
	} else {
		resp.Body.Close()
	}

	// This should fail.
	if _, err := client.Post(ts.URL, "application/octet-stream", bytes.NewReader(make([]byte, 100))); err == nil {
		t.Fatal("expected error")
	}
}
