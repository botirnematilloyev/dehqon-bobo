package notifier

import (
	"context"
	"log/slog"
)

type LogNotifier struct{}

func NewLogNotifier() *LogNotifier {
	return &LogNotifier{}
}

func (n *LogNotifier) Send(ctx context.Context, identifier string, code string) error {
	slog.Warn("DEV ONLY: OTP code",
		"identifier", identifier,
		"code", code,
	)
	return nil
}
