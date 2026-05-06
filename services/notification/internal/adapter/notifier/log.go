// Package notifier holds Notifier adapters. For now we ship just a Log
// adapter (writes a structured line via zerolog); add Email/FCM/etc by
// implementing the port.Notifier interface and registering them in the
// composition root.
package notifier

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/liliksetyawan/orderflow/services/notification/internal/app/port"
	"github.com/liliksetyawan/orderflow/services/notification/internal/domain"
)

type Log struct {
	log zerolog.Logger
}

func NewLog(log zerolog.Logger) *Log {
	return &Log{log: log.With().Str("notifier", "log").Logger()}
}

var _ port.Notifier = (*Log)(nil)

func (l *Log) Channel() string { return "log" }

func (l *Log) Send(ctx context.Context, n domain.Notification, reason string) error {
	ev := l.log.Info().
		Str("order_id", n.OrderID).
		Str("customer_id", n.CustomerID).
		Str("type", n.Type)
	if reason != "" {
		ev = ev.Str("reason", reason)
	}
	ev.Msg("NOTIFICATION SENT")
	return nil
}
