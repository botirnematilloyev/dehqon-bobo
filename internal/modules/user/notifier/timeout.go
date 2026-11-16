package notifier

import (
	"context"
	"time"
)

type Notifier interface {
	Send(ctx context.Context, identifier, code string) error
}

type timeoutNotifier struct {
	inner   Notifier
	timeout time.Duration
}

func WithTimeout(inner Notifier, timeout time.Duration) Notifier {
	if timeout <= 0 {
		return inner
	}
	return &timeoutNotifier{inner: inner, timeout: timeout}
}

func (n *timeoutNotifier) Send(ctx context.Context, identifier, code string) error {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	return n.inner.Send(ctx, identifier, code)
}
