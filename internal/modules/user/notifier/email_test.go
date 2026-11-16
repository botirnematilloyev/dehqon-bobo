package notifier

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func testEmailNotifier() *EmailNotifier {
	return NewEmailNotifier(EmailConfig{
		Host: "smtp.example.com",
		Port: "587",
		From: "noreply@dehqonbobo.uz",
	})
}

func TestEmailNotifier_message(t *testing.T) {
	t.Run("builds a well-formed message", func(t *testing.T) {
		msg, err := testEmailNotifier().message("user@example.com", "123456")
		require.NoError(t, err)

		s := string(msg)
		require.Contains(t, s, "From: noreply@dehqonbobo.uz\r\n")
		require.Contains(t, s, "To: user@example.com\r\n")
		require.Contains(t, s, "Subject: Tasdiqlash kodi\r\n")
		require.Contains(t, s, "123456")

		headers, body, found := strings.Cut(s, "\r\n\r\n")
		require.True(t, found, "headers and body must be separated by a blank line")
		require.NotContains(t, headers, "123456", "code belongs in the body")
		require.Contains(t, body, "123456")
	})

	t.Run("uses CRLF line endings", func(t *testing.T) {
		msg, err := testEmailNotifier().message("user@example.com", "123456")
		require.NoError(t, err)

		s := string(msg)
		require.NotContains(t, strings.ReplaceAll(s, "\r\n", ""), "\n")
	})

	t.Run("parses back as multipart/alternative with both variants", func(t *testing.T) {
		msg, err := testEmailNotifier().message("user@example.com", "422217")
		require.NoError(t, err)

		parsed, err := mail.ReadMessage(bytes.NewReader(msg))
		require.NoError(t, err)

		mediaType, params, err := mime.ParseMediaType(parsed.Header.Get("Content-Type"))
		require.NoError(t, err)
		require.Equal(t, "multipart/alternative", mediaType)
		require.NotEmpty(t, params["boundary"])

		require.Equal(t, 2, strings.Count(string(msg), "Content-Transfer-Encoding: quoted-printable"))

		bodies := map[string]string{}
		mr := multipart.NewReader(parsed.Body, params["boundary"])
		for {
			part, err := mr.NextPart()
			if errors.Is(err, io.EOF) {
				break
			}
			require.NoError(t, err)

			ct, _, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
			require.NoError(t, err)

			decoded, err := io.ReadAll(part)
			require.NoError(t, err)
			bodies[ct] = string(decoded)
		}

		require.Len(t, bodies, 2, "both a text and an HTML variant must be present")
		require.Contains(t, bodies["text/plain"], "Tasdiqlash kodingiz: 422217")
		require.Contains(t, bodies["text/html"], "422217")
		require.Contains(t, bodies["text/html"], "<!DOCTYPE html>")
		require.NotContains(t, bodies["text/html"], "=\r\n",
			"quoted-printable soft breaks must be gone after decoding")
	})

	t.Run("keeps the html variant transport safe", func(t *testing.T) {
		msg, err := testEmailNotifier().message("user@example.com", "422217")
		require.NoError(t, err)

		for _, line := range strings.Split(string(msg), "\r\n") {
			require.LessOrEqual(t, len(line), 998, "RFC 5322 caps a line at 998 octets")
		}
	})

	t.Run("adds the display name when configured", func(t *testing.T) {
		n := NewEmailNotifier(EmailConfig{
			Host:     "smtp.example.com",
			Port:     "587",
			From:     "noreply@dehqonbobo.uz",
			FromName: "Dehqon",
		})

		msg, err := n.message("user@example.com", "123456")
		require.NoError(t, err)
		require.Contains(t, string(msg), "From: Dehqon <noreply@dehqonbobo.uz>\r\n")
	})

	t.Run("encodes a non-ASCII display name", func(t *testing.T) {
		n := NewEmailNotifier(EmailConfig{
			Host:     "smtp.example.com",
			Port:     "587",
			From:     "noreply@dehqonbobo.uz",
			FromName: "Dehqon — Kirish",
		})

		msg, err := n.message("user@example.com", "123456")
		require.NoError(t, err)

		s := string(msg)
		require.Contains(t, s, "<noreply@dehqonbobo.uz>")
		require.Contains(t, s, "=?utf-8?q?", "non-ASCII name must be RFC 2047 encoded")
		for i := 0; i < len(s); i++ {
			require.Less(t, s[i], byte(0x80), "message must stay seven-bit clean")
		}
	})

	t.Run("neutralises a display name carrying a line break", func(t *testing.T) {
		n := NewEmailNotifier(EmailConfig{
			Host:     "smtp.example.com",
			Port:     "587",
			From:     "noreply@dehqonbobo.uz",
			FromName: "Dehqon\r\nBcc: victim@example.com",
		})

		msg, err := n.message("user@example.com", "123456")
		require.NoError(t, err)

		headers, _, _ := strings.Cut(string(msg), "\r\n\r\n")
		from, _, _ := strings.Cut(headers, "\r\n")
		require.Equal(t, "From: =?utf-8?q?Dehqon=0D=0ABcc:_victim@example.com?= <noreply@dehqonbobo.uz>", from,
			"the break must be encoded, not emitted as a new header line")
		require.NotContains(t, strings.SplitN(headers, "\r\n", 2)[1], "Bcc:")
	})

	t.Run("rejects header injection", func(t *testing.T) {
		cases := map[string]string{
			"bcc injection": "user@example.com\r\nBcc: victim@example.com",
			"bare newline":  "user@example.com\nSubject: hijacked",
			"bare carriage": "user@example.com\rX-Evil: 1",
		}

		for name, to := range cases {
			t.Run(name, func(t *testing.T) {
				_, err := testEmailNotifier().message(to, "123456")
				require.Error(t, err)
			})
		}
	})
}
