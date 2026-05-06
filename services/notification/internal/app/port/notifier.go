package port

import (
	"context"

	"github.com/lilik-setyawan/orderflow/services/notification/internal/domain"
)

// Notifier is the outbound port for delivering a notification. Each
// concrete adapter (log, email, push) reports the channel name it serves
// so the use case can persist the send under the right channel for
// idempotency. reason carries the cancellation reason for OrderCanceled
// notifications and is empty otherwise.
type Notifier interface {
	Channel() string
	Send(ctx context.Context, n domain.Notification, reason string) error
}
