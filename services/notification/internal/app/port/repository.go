// Package port defines the interfaces (ports) the notification application
// depends on. Driven adapters under internal/adapter implement these.
package port

import (
	"context"

	"github.com/lilik-setyawan/orderflow/services/notification/internal/domain"
)

// NotificationRepository persists a record per *successful* send. Create
// surfaces the UNIQUE(order_id, type, channel) collision as
// domain.ErrAlreadyExists, which the use case treats as idempotent.
type NotificationRepository interface {
	Create(ctx context.Context, n *domain.Notification) error
	GetByOrderTypeChannel(ctx context.Context, orderID, typ, channel string) (*domain.Notification, error)
}
