package notifier

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type stubNotifier struct {
	delay time.Duration
	err   error

	gotIdentifier string
	gotCode       string
	gotDeadline   time.Time
	hadDeadline   bool
}

func (s *stubNotifier) Send(ctx context.Context, identifier, code string) error {
	s.gotIdentifier = identifier
	s.gotCode = code
	s.gotDeadline, s.hadDeadline = ctx.Deadline()

	if s.delay > 0 {
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return s.err
}

func TestWithTimeout(t *testing.T) {
	ctx := context.Background()

	t.Run("passes the call through", func(t *testing.T) {
		stub := &stubNotifier{}

		require.NoError(t, WithTimeout(stub, time.Second).Send(ctx, "555", "123456"))
		require.Equal(t, "555", stub.gotIdentifier)
		require.Equal(t, "123456", stub.gotCode)
	})

	t.Run("gives the inner notifier a deadline", func(t *testing.T) {
		stub := &stubNotifier{}

		require.NoError(t, WithTimeout(stub, time.Second).Send(ctx, "555", "123456"))
		require.True(t, stub.hadDeadline, "inner notifier must receive a deadline")
	})

	t.Run("times out a slow notifier", func(t *testing.T) {
		stub := &stubNotifier{delay: time.Second}

		err := WithTimeout(stub, 50*time.Millisecond).Send(ctx, "555", "123456")
		require.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("propagates the inner error", func(t *testing.T) {
		want := errors.New("smtp is down")
		stub := &stubNotifier{err: want}

		err := WithTimeout(stub, time.Second).Send(ctx, "555", "123456")
		require.ErrorIs(t, err, want)
	})

	t.Run("does not extend a shorter parent deadline", func(t *testing.T) {
		stub := &stubNotifier{}

		parent, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		require.NoError(t, WithTimeout(stub, time.Hour).Send(parent, "555", "123456"))
		require.Less(t, time.Until(stub.gotDeadline), time.Second)
	})

	t.Run("returns the inner notifier when the timeout is zero", func(t *testing.T) {
		stub := &stubNotifier{}
		require.Same(t, Notifier(stub), WithTimeout(stub, 0))
	})
}
