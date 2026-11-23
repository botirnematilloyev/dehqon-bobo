package notifier

import (
	"mime"
	"strings"
	"testing"

	"dehqon-bobo/internal/platform/i18n"

	"github.com/stretchr/testify/require"
)

func TestOTPMessagesLocalized(t *testing.T) {
	for _, k := range []i18n.Key{msgSubject, msgYourCode, msgIntro, msgValidFor, msgNeverShare, msgIgnoreIfNot} {
		require.True(t, i18n.Registered(k), k)
		for _, l := range []i18n.Locale{i18n.Uzbek, i18n.Russian} {
			require.NotEqual(t, string(k), i18n.T(l, k))
		}
	}

	msg, err := testEmailNotifier().messageIn(i18n.Russian, "user@example.com", "422217")
	require.NoError(t, err)
	headers, body, _ := strings.Cut(string(msg), "\r\n\r\n")

	var subject string
	for _, line := range strings.Split(headers, "\r\n") {
		if v, ok := strings.CutPrefix(line, "Subject: "); ok {
			subject = v
		}
	}
	require.NotContains(t, subject, "Код", "kirill sarlavha RFC 2047 bilan kodlanadi")
	decoded, err := new(mime.WordDecoder).DecodeHeader(subject)
	require.NoError(t, err)
	require.Equal(t, "Код подтверждения", decoded)
	require.Contains(t, body, "422217")
	require.Contains(t, body, `lang=3D"ru"`, "html lang (quoted-printable)")

	require.Equal(t, "Ваш код подтверждения: 422217\n\nНикому не сообщайте код — наши сотрудники никогда его не спрашивают.",
		otpText(i18n.Russian, "422217"))
	require.Equal(t, "Тасдиқлаш кодингиз: 422217\n\nКодни ҳеч кимга берманг — ходимларимиз ҳам ҳеч қачон сўрамайди.",
		otpText(i18n.UzbekCyrillic, "422217"))
}
