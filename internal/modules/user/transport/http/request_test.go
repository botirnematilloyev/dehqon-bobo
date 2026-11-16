package http

import (
	"dehqon-bobo/internal/modules/user/domain"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseProvider(t *testing.T) {
	t.Run("known providers", func(t *testing.T) {
		cases := map[string]domain.AuthProvider{
			"email":    domain.ProviderEmail,
			"telegram": domain.ProviderTelegram,
		}

		for in, want := range cases {
			t.Run(in, func(t *testing.T) {
				got, err := parseProvider(in)
				require.NoError(t, err)
				require.Equal(t, want, got)
			})
		}
	})

	t.Run("rejects everything else", func(t *testing.T) {

		cases := []string{"", "Email", "EMAIL", " email", "email ", "sms", "e-mail", "phone"}

		for _, in := range cases {
			t.Run(in, func(t *testing.T) {
				got, err := parseProvider(in)
				require.ErrorIs(t, err, domain.ErrUnsupportedProvider)
				require.Zero(t, got)
			})
		}
	})
}

func TestParsePlatform(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		require.Nil(t, parsePlatform(nil))
	})

	t.Run("known platforms", func(t *testing.T) {
		cases := map[string]domain.Platform{
			"web":     domain.PlatformWeb,
			"ios":     domain.PlatformIOS,
			"android": domain.PlatformAndroid,
		}

		for in, want := range cases {
			t.Run(in, func(t *testing.T) {
				got := parsePlatform(&in)
				require.NotNil(t, got)
				require.Equal(t, want, *got)
			})
		}
	})

	t.Run("unknown platform returns nil, not an error", func(t *testing.T) {
		for _, in := range []string{"", "Windows", "linux", "iOS"} {
			t.Run(in, func(t *testing.T) {
				require.Nil(t, parsePlatform(&in))
			})
		}
	})
}

func TestRequestOTPRequest_Validate(t *testing.T) {
	t.Run("accepts non-empty identifier", func(t *testing.T) {
		req := RequestOTPRequest{Provider: "email", Identifier: "a@b.uz"}
		require.NoError(t, req.Validate())
	})

	t.Run("rejects empty identifier", func(t *testing.T) {
		req := RequestOTPRequest{Provider: "email"}
		require.Error(t, req.Validate())
	})

	t.Run("does not validate provider", func(t *testing.T) {
		req := RequestOTPRequest{Provider: "carrier-pigeon", Identifier: "a@b.uz"}
		require.NoError(t, req.Validate())
	})
}

func TestVerifyOTPRequest_Validate(t *testing.T) {
	valid := func() VerifyOTPRequest {
		return VerifyOTPRequest{Provider: "email", Identifier: "a@b.uz", Code: "123456"}
	}

	t.Run("accepts a valid request", func(t *testing.T) {
		require.NoError(t, valid().Validate())
	})

	t.Run("rejects codes that are not exactly 6 characters", func(t *testing.T) {
		for _, code := range []string{"", "1", "12345", "1234567"} {
			t.Run(code, func(t *testing.T) {
				req := valid()
				req.Code = code
				require.Error(t, req.Validate())
			})
		}
	})

	t.Run("rejects empty identifier", func(t *testing.T) {
		req := valid()
		req.Identifier = ""
		require.Error(t, req.Validate())
	})
}

func TestRefreshRequest_Validate(t *testing.T) {
	t.Run("accepts non-empty token", func(t *testing.T) {
		require.NoError(t, RefreshRequest{RefreshToken: "abc"}.Validate())
	})

	t.Run("rejects empty token", func(t *testing.T) {
		require.Error(t, RefreshRequest{}.Validate())
	})
}
