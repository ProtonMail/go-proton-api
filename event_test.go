package proton_test

import (
	"context"
	"testing"
	"time"

	"github.com/ProtonMail/go-proton-api"
	"github.com/ProtonMail/go-proton-api/server"
	"github.com/stretchr/testify/require"
)

func TestEventStreamer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := server.New()
	defer s.Close()

	m := proton.New(
		proton.WithHostURL(s.GetHostURL()),
		proton.WithTransport(proton.InsecureTransport()),
	)

	_, _, err := s.CreateUser("username", "email@pm.me", []byte("password"))
	require.NoError(t, err)

	c, _, err := m.NewClientWithLogin(ctx, "username", []byte("password"))
	require.NoError(t, err)

	createTestMessages(t, c, "password", 10)

	latestEventID, err := c.GetLatestEventID(ctx)
	require.NoError(t, err)

	eventCh := make(chan proton.Event)

	go func() {
		for event := range c.NewEventStream(ctx, time.Second, 0, latestEventID) {
			eventCh <- event
		}
	}()

	// Perform some action to generate an event.
	metadata, err := c.GetMessageMetadata(ctx, proton.MessageFilter{})
	require.NoError(t, err)
	require.NoError(t, c.LabelMessages(ctx, []string{metadata[0].ID}, proton.TrashLabel))

	// Wait for the first event.
	<-eventCh

	// Close the client; this should stop the client's event streamer.
	c.Close()

	// Create a new client and perform some actions with it to generate more events.
	cc, _, err := m.NewClientWithLogin(ctx, "username", []byte("password"))
	require.NoError(t, err)
	defer cc.Close()

	require.NoError(t, cc.LabelMessages(ctx, []string{metadata[1].ID}, proton.TrashLabel))

	// We should not receive any more events from the original client.
	select {
	case <-eventCh:
		require.Fail(t, "received unexpected event")

	default:
		// ...
	}
}
