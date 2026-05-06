// Package port defines the interfaces (ports) the payment application layer
// depends on. Driven adapters under internal/adapter implement these.
package port

import (
	"context"

	"github.com/liliksetyawan/orderflow/services/payment/internal/domain"
)

// PaymentRepository is the persistence port for the Payment aggregate.
//
// Create returns domain.ErrAlreadyExists when a payment for the same
// order_id already exists — the caller should treat this as success
// (idempotent retry).
type PaymentRepository interface {
	Create(ctx context.Context, p *domain.Payment, ev []domain.Event) error
	Save(ctx context.Context, p *domain.Payment, ev []domain.Event) error
	GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
}
