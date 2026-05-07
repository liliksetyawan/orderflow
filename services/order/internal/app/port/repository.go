// Package port defines the interfaces (ports) the application layer depends
// on. Driven adapters (postgres, uuid, ...) live under internal/adapter and
// implement these interfaces. The application — domain + use cases — never
// imports a concrete adapter, only ports.
package port

import (
	"context"

	"github.com/liliksetyawan/orderflow/services/order/internal/domain"
)

// OrderRepository is the persistence port for the Order aggregate.
//
// Create / Save accept domain events alongside the entity so the adapter can
// persist them in the *same* transaction as the state change. That's the
// transactional outbox guarantee — no entity write without its events, no
// events without their entity write.
type OrderRepository interface {
	Create(ctx context.Context, o *domain.Order, ev []domain.Event) error
	Save(ctx context.Context, o *domain.Order, ev []domain.Event) error
	Get(ctx context.Context, id string) (*domain.Order, error)

	// List returns orders ordered by created_at DESC (newest first), plus
	// the total row count for the same filter (for pagination). Items are
	// not loaded — the list view doesn't need them and skipping the join
	// keeps the query cheap. status="" means no filter.
	List(ctx context.Context, status string, limit, offset int) (orders []*domain.Order, total int, err error)
}
