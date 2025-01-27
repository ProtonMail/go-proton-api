package proton_test

import (
	"bytes"
	"context"
	"mime"
	"mime/multipart"
	"testing"

	"github.com/henrybear327/go-proton-api"
	"github.com/henrybear327/go-proton-api/server"
	"github.com/stretchr/testify/require"
)

func TestReportBug(t *testing.T) {
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

	res, err := m.ReportBug(context.Background(), proton.ReportBugReq{
		OS:         "linux",
		OSVersion:  "5.4.0-42-generic",
		Browser:    "firefox",
		ClientType: proton.ClientTypeEmail,
	})
	require.NoError(t, err)
	require.Equal(t, (*string)(nil), res.Token)

	mimeType, mimeParams, err := mime.ParseMediaType(calls[0].RequestHeader.Get("Content-Type"))
	require.NoError(t, err)
	require.Equal(t, "multipart/form-data", mimeType)

	form, err := multipart.NewReader(bytes.NewReader(calls[0].RequestBody), mimeParams["boundary"]).ReadForm(0)
	require.NoError(t, err)

	require.Len(t, form.Value, 5)
	require.Equal(t, "linux", form.Value["OS"][0])
	require.Equal(t, "5.4.0-42-generic", form.Value["OSVersion"][0])
	require.Equal(t, "firefox", form.Value["Browser"][0])
	require.Equal(t, "1", form.Value["ClientType"][0])
	require.Equal(t, "0", form.Value["AsyncAttachments"][0])
}

func TestReportBugAsync(t *testing.T) {
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

	res, err := m.ReportBug(context.Background(), proton.ReportBugReq{
		OS:               "linux",
		OSVersion:        "5.4.0-42-generic",
		Browser:          "firefox",
		ClientType:       proton.ClientTypeEmail,
		AsyncAttachments: proton.AttachmentTypeAsync,
	})
	require.NoError(t, err)
	require.NotEmpty(t, res.Token)

	mimeType, mimeParams, err := mime.ParseMediaType(calls[0].RequestHeader.Get("Content-Type"))
	require.NoError(t, err)
	require.Equal(t, "multipart/form-data", mimeType)

	form, err := multipart.NewReader(bytes.NewReader(calls[0].RequestBody), mimeParams["boundary"]).ReadForm(0)
	require.NoError(t, err)

	require.Len(t, form.Value, 5)
	require.Equal(t, "linux", form.Value["OS"][0])
	require.Equal(t, "5.4.0-42-generic", form.Value["OSVersion"][0])
	require.Equal(t, "firefox", form.Value["Browser"][0])
	require.Equal(t, "1", form.Value["ClientType"][0])
	require.Equal(t, "1", form.Value["AsyncAttachments"][0])

	err = m.ReportBugAttachement(context.Background(), proton.ReportBugAttachmentReq{
		Product: proton.ClientTypeEmail,
		Body:    "Comment without any attachment",
		Token:   *res.Token,
	})

	require.NoError(t, err)

	err = m.ReportBugAttachement(context.Background(), proton.ReportBugAttachmentReq{
		Product: proton.ClientTypeEmail,
		Body:    "Comment without any attachment",
		Token:   "not a good token",
	})

	require.Error(t, err)
}
