package proton_test

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"sync"
	"testing"

	"github.com/ProtonMail/go-proton-api"
	"github.com/ProtonMail/go-proton-api/server"
	"github.com/stretchr/testify/require"
)

type retryTestTransport struct {
	mu       sync.Mutex
	attempts map[string]int
	path     string
}

func newRetryTestTransport(path string) *retryTestTransport {
	return &retryTestTransport{
		attempts: make(map[string]int),
		path:     path,
	}
}

func (t *retryTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	key := req.URL.Path + req.URL.RawQuery
	t.attempts[key]++
	attempt := t.attempts[key]
	t.mu.Unlock()

	if attempt == 1 {
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header:     http.Header{"Retry-After": []string{"0"}},
			Body:       io.NopCloser(bytes.NewReader([]byte("rate limited"))),
		}, nil
	}

	return proton.InsecureTransport().RoundTrip(req)
}

func (t *retryTestTransport) AttemptCount(path string) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.attempts[path]
}

func TestReportBugMultipartRetryWithAttachment(t *testing.T) {
	s := server.New()
	defer s.Close()

	originalBody := []byte("this is a non-trivial test attachment body that is definitely longer than a few bytes and contains various characters to ensure byte-accurate comparison works correctly 0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ!@#$%^&*()[]{}|;':\",./<>?`~")

	var capturedServerCalls []server.Call
	s.AddCallWatcher(func(call server.Call) {
		capturedServerCalls = append(capturedServerCalls, call)
	})

	transport := newRetryTestTransport("/core/v4/reports/bug")

	m := proton.New(
		proton.WithHostURL(s.GetHostURL()),
		proton.WithTransport(transport),
		proton.WithRetryCount(3),
	)
	defer m.Close()

	_, err := m.ReportBug(context.Background(), proton.ReportBugReq{
		OS:         "linux",
		OSVersion:  "5.4.0-42-generic",
		Browser:    "firefox",
		ClientType: proton.ClientTypeEmail,
	}, proton.ReportBugAttachment{
		Name:     "test-attachment",
		Filename: "test.txt",
		MIMEType: "text/plain",
		Body:     originalBody,
	})
	require.NoError(t, err)

	require.Len(t, capturedServerCalls, 1, "expected exactly 1 call to server (the retry)")
	require.Equal(t, http.StatusOK, capturedServerCalls[0].Status)

	require.Equal(t, 2, transport.AttemptCount("/core/v4/reports/bug"), "expected exactly 2 attempts through transport: one 429, one success")

	form, err := multipart.NewReader(
		bytes.NewReader(capturedServerCalls[0].RequestBody),
		mustParseBoundary(t, capturedServerCalls[0].RequestHeader.Get("Content-Type")),
	).ReadForm(0)
	require.NoError(t, err, "failed to parse multipart form from retry body")

	require.Contains(t, form.File, "test-attachment", "attachment field missing from retry body")
	attachmentFile, err := form.File["test-attachment"][0].Open()
	require.NoError(t, err)
	attachmentBody, err := io.ReadAll(attachmentFile)
	require.NoError(t, attachmentFile.Close())
	require.NoError(t, err)

	require.Equal(t, originalBody, attachmentBody, "attachment body mismatch on retry: SetRetryResetReaders did not properly reset the reader")
}

func mustParseBoundary(t *testing.T, contentType string) string {
	_, params, err := mime.ParseMediaType(contentType)
	require.NoError(t, err, "failed to parse Content-Type")
	boundary, ok := params["boundary"]
	require.True(t, ok, "Content-Type missing boundary")
	return boundary
}

func TestReportBugMultipart(t *testing.T) {
	s := server.New()
	defer s.Close()

	m := proton.New(
		proton.WithHostURL(s.GetHostURL()),
		proton.WithTransport(proton.InsecureTransport()),
	)
	defer m.Close()

	var calls []server.Call

	s.AddCallWatcher(func(call server.Call) {
		calls = append(calls, call)
	})

	_, err := m.ReportBug(context.Background(), proton.ReportBugReq{
		OS:         "linux",
		OSVersion:  "5.4.0-42-generic",
		Browser:    "firefox",
		ClientType: proton.ClientTypeEmail,
	})
	require.NoError(t, err)

	require.Len(t, calls, 1, "expected exactly 1 call (no retries)")

	mimeType, mimeParams, err := mime.ParseMediaType(calls[0].RequestHeader.Get("Content-Type"))
	require.NoError(t, err)
	require.Equal(t, "multipart/form-data", mimeType)

	form, err := multipart.NewReader(bytes.NewReader(calls[0].RequestBody), mimeParams["boundary"]).ReadForm(0)
	require.NoError(t, err)

	require.Len(t, form.Value, 5)
	require.Equal(t, "linux", form.Value["OS"][0])
}
