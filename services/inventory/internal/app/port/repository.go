// Package port defines the interfaces (ports) the inventory application
// layer depends on.
package port

import (
	"context"

	"github.com/liliksetyawan/orderflow/services/inventory/internal/domain"
)

// ReservationRepository is the persistence port for the Reservation
// aggregate. The three "atomic" methods each wrap their work plus outbox
// rows in a single transaction, which is what gives the saga its
// at-least-once-with-no-duplicates property.
type ReservationRepository interface {
	// ReserveAtomic inserts a RESERVED reservation, decrements stock for
	// each item with a conditional UPDATE, and appends events. Returns:
	//   domain.ErrAlreadyExists     — UNIQUE(order_id) collision
	//   domain.ErrInsufficientStock — any item couldn't be satisfied
	ReserveAtomic(ctx context.Context, r *domain.Reservation, ev []domain.Event) error

	// RecordFailed inserts a FAILED reservation + events (no stock changes).
	// Used when ReserveAtomic returned ErrInsufficientStock.
	RecordFailed(ctx context.Context, r *domain.Reservation, ev []domain.Event) error

	// ReleaseAtomic moves a RESERVED reservation to RELEASED and increments
	// stocks back to their pre-reservation levels. Optimistic-locked.
	ReleaseAtomic(ctx context.Context, r *domain.Reservation, ev []domain.Event) error

	GetByOrderID(ctx context.Context, orderID string) (*domain.Reservation, error)
}
