package proton_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/ProtonMail/gluon/rfc822"
	"github.com/bradenaw/juniper/stream"
	"github.com/henrybear327/go-proton-api"
	"github.com/henrybear327/go-proton-api/server"
	"github.com/stretchr/testify/require"
)

func Test_chunkSized(t *testing.T) {
	type args struct {
		vals    []int
		maxLen  int
		maxSize int
		getSize func(int) int
	}

	tests := []struct {
		name string
		args args
		want [][]int
	}{
		{
			name: "limit by length",
			args: args{
				vals:    []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
				maxLen:  3, // Split into chunks of at most 3
				maxSize: 100,
				getSize: func(i int) int { return i },
			},
			want: [][]int{
				{1, 2, 3},
				{4, 5, 6},
				{7, 8, 9},
				{10},
			},
		},
		{
			name: "limit by size",
			args: args{
				vals:    []int{1, 1, 1, 1, 1, 2, 2, 2, 2, 2},
				maxLen:  100,
				maxSize: 5, // Split into chunks of at most 5
				getSize: func(i int) int { return i },
			},
			want: [][]int{
				{1, 1, 1, 1, 1},
				{2, 2},
				{2, 2},
				{2},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := proton.ChunkSized(tt.args.vals, tt.args.maxLen, tt.args.maxSize, tt.args.getSize); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ChunkSized() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageImport_RelatedInlinePlaintext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := server.New()
	defer s.Close()

	m := proton.New(
		proton.WithHostURL(s.GetHostURL()),
		proton.WithTransport(proton.InsecureTransport()),
	)

	_, _, err := s.CreateUser("user", []byte("pass"))
	require.NoError(t, err)

	c, _, err := m.NewClientWithLogin(ctx, "user", []byte("pass"))
	require.NoError(t, err)

	tests := []struct {
		name    string
		literal string
		want    rfc822.MIMEType
	}{
		{
			name: "RelatedInlinePlaintext",
			literal: "From: Nathaniel Borenstein <nsb@bellcore.com>\n" +
				"To:  Ned Freed <ned@innosoft.com>\n" +
				"Subject: Sample message (import inline)\n" +
				"MIME-Version: 1.0\nContent-type: multipart/related; boundary=\"BOUNDARY\"\n\n" +
				"--BOUNDARY\nContent-type: text/plain; charset=us-ascii\n\n" +
				"Hello world\n" +
				"--BOUNDARY\n" +
				"Content-Type: image/gif; name=\"email-action-left.gif\"\n" +
				"Content-Transfer-Encoding: base64\n" +
				"Content-ID: <part1.D96BFAE9.E2E1CAE3@protonmail.com>\n" +
				"Content-Disposition: inline; filename=\"email-action-left.gif\"\n\n" +
				"SGVsbG8gQXR0YWNobWVudA==\n" +
				"--BOUNDARY--",
			want: rfc822.TextPlain,
		},
		{
			name: "RelatedInlineHTML",
			literal: "From: Bridge Second Test <bridge_second@test.com>\n" +
				"To: Bridge Test <bridge@test.com>\n" +
				"Subject: Html Inline Importing\n" +
				"Content-Type: multipart/related; boundary=\"61FA22A41A3F46E8E90EF528\"\n\n" +
				"This is a multi-part message in MIME format.\n" +
				"--61FA22A41A3F46E8E90EF528\n" +
				"Content-Type: text/html; charset=utf-8\n" +
				"Content-Transfer-Encoding: 7bit\n\n" +
				"<html>\n<head>\n<meta http-equiv=\"content-type\" content=\"text/html; charset=UTF-8\">\n</head>\n" +
				"<body text=\"#000000\" bgcolor=\"#FFFFFF\">\n" +
				"<p><br>\n</p>\n<p>Behold! An inline <img moz-do-not-send=\"false\"\nsrc=\"cid:part1.D96BFAE9.E2E1CAE3@protonmail.com\" alt=\"\"\nwidth=\"24\" height=\"24\"><br>\n</p>\n" +
				"</body>\n</html>\n\n" +
				"--61FA22A41A3F46E8E90EF528\n" +
				"Content-Type: image/gif; name=\"email-action-left.gif\"\n" +
				"Content-Transfer-Encoding: base64\nContent-ID: <part1.D96BFAE9.E2E1CAE3@protonmail.com>\n" +
				"Content-Disposition: inline; filename=\"email-action-left.gif\"\n\n" +
				"R0lGODlhGAAYANUAACcsKOHs4kppTH6tgYWxiIq0jTVENpG5lDI/M7bRuEaJSkqOTk2RUU+P\nU16lYl+lY2iva262cXS6d3rDfYLNhWeeamKTZGSVZkNbRGqhbOPt4////+7u7qioqFZWVlNT\nUyIiIgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA\nAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACwAAAAAGAAYAAAG\n/8CNcLjRJAqVRqNSSGiI0GFgoKhar4NAdHioMhyRCYUyiTgY1cOWUH1ILgIDAGAQXCSPKgHa\nXUAyGCCCg4IYGRALCmpCAVUQFgiEkiAIFhBVWhtUDxmRk5IIGXkDRQoMEoGfHpIYEmhGCg4X\nnyAdHB+SFw4KRwoRArQdG7eEAhEKSAoTBoIdzs/Cw7iCBhMKSQoUAIJbQ8QgABQKStnbIN1C\n3+HjFcrMtdDO6dMg1dcFvsCfwt+CxsgJYs3a10+QLl4aTKGitYpQq1eaFHDyREtQqFGMHEGq\nSMkSJi4K/ACiZQiRIihsJL6JM6fOnTwK9kTpYgqMGDJm0JzsNuWKTw0FWdANMYJECRMnW4IA\nADs=\n\n" +
				"--61FA22A41A3F46E8E90EF528--",
			want: rfc822.TextHTML,
		},
		{
			name: "Multipart/mixed single part plain",
			literal: "From: Nathaniel Borenstein <nsb@bellcore.com>\n" +
				"To:  Ned Freed <ned@innosoft.com>\n" +
				"Subject: Sample message (import inline)\n" +
				"MIME-Version: 1.0\nContent-type: multipart/mixed; boundary=\"BOUNDARY\"\n\n" +
				"--BOUNDARY\nContent-type: text/plain; charset=us-ascii\n\n" +
				"Hello world\n" +
				"--BOUNDARY\n" +
				"Content-Type: image/gif; name=\"email-action-left.gif\"\n" +
				"Content-Transfer-Encoding: base64\n" +
				"Content-ID: <part1.D96BFAE9.E2E1CAE3@protonmail.com>\n" +
				"Content-Disposition: inline; filename=\"email-action-left.gif\"\n\n" +
				"SGVsbG8gQXR0YWNobWVudA==\n" +
				"--BOUNDARY--",
			want: rfc822.TextPlain,
		},
		{
			name: "Multipart/mixed single part HTML",
			literal: "From: Bridge Second Test <bridge_second@test.com>\n" +
				"To: Bridge Test <bridge@test.com>\n" +
				"Subject: Html Inline Importing\n" +
				"Content-Type: multipart/mixed; boundary=\"61FA22A41A3F46E8E90EF528\"\n\n" +
				"This is a multi-part message in MIME format.\n" +
				"--61FA22A41A3F46E8E90EF528\n" +
				"Content-Type: text/html; charset=utf-8\n" +
				"Content-Transfer-Encoding: 7bit\n\n" +
				"<html>\n<head>\n<meta http-equiv=\"content-type\" content=\"text/html; charset=UTF-8\">\n</head>\n" +
				"<body text=\"#000000\" bgcolor=\"#FFFFFF\">\n" +
				"<p><br>\n</p>\n<p>Behold! An inline <img moz-do-not-send=\"false\"\nsrc=\"cid:part1.D96BFAE9.E2E1CAE3@protonmail.com\" alt=\"\"\nwidth=\"24\" height=\"24\"><br>\n</p>\n" +
				"</body>\n</html>\n\n" +
				"--61FA22A41A3F46E8E90EF528\n" +
				"Content-Type: image/gif; name=\"email-action-left.gif\"\n" +
				"Content-Transfer-Encoding: base64\nContent-ID: <part1.D96BFAE9.E2E1CAE3@protonmail.com>\n" +
				"Content-Disposition: inline; filename=\"email-action-left.gif\"\n\n" +
				"R0lGODlhGAAYANUAACcsKOHs4kppTH6tgYWxiIq0jTVENpG5lDI/M7bRuEaJSkqOTk2RUU+P\nU16lYl+lY2iva262cXS6d3rDfYLNhWeeamKTZGSVZkNbRGqhbOPt4////+7u7qioqFZWVlNT\nUyIiIgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA\nAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACwAAAAAGAAYAAAG\n/8CNcLjRJAqVRqNSSGiI0GFgoKhar4NAdHioMhyRCYUyiTgY1cOWUH1ILgIDAGAQXCSPKgHa\nXUAyGCCCg4IYGRALCmpCAVUQFgiEkiAIFhBVWhtUDxmRk5IIGXkDRQoMEoGfHpIYEmhGCg4X\nnyAdHB+SFw4KRwoRArQdG7eEAhEKSAoTBoIdzs/Cw7iCBhMKSQoUAIJbQ8QgABQKStnbIN1C\n3+HjFcrMtdDO6dMg1dcFvsCfwt+CxsgJYs3a10+QLl4aTKGitYpQq1eaFHDyREtQqFGMHEGq\nSMkSJi4K/ACiZQiRIihsJL6JM6fOnTwK9kTpYgqMGDJm0JzsNuWKTw0FWdANMYJECRMnW4IA\nADs=\n\n" +
				"--61FA22A41A3F46E8E90EF528--",
			want: rfc822.TextHTML,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			str, err := importMessage(t, c, ctx, "pass", tt.literal)
			require.NoError(t, err)

			// check import status
			res, err := stream.Collect(ctx, str)
			require.NoError(t, err)
			require.Equal(t, 1, len(res))
			require.NotEmpty(t, res[0].MessageID)

			// check message imported
			full, err := c.GetFullMessage(ctx, res[0].MessageID, proton.NewSequentialScheduler(), proton.NewDefaultAttachmentAllocator())
			require.NoError(t, err)

			require.Equal(t, tt.want, full.MIMEType)
		})
	}

}
