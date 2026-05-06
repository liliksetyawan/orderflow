// Package domain holds the Payment aggregate and its business rules. Pure
// (zero infra deps): no pgx, no rabbitmq, no zerolog. The state machine is:
//
//	PENDING ─┬──> AUTHORIZED ──> RELEASED
//	         └──> FAILED
package domain

import "time"

type PaymentStatus string

const (
	StatusPending    PaymentStatus = "PENDING"
	StatusAuthorized PaymentStatus = "AUTHORIZED"
	StatusFailed     PaymentStatus = "FAILED"
	StatusReleased   PaymentStatus = "RELEASED"
)

type Payment struct {
	ID          string
	OrderID     string
	CustomerID  string
	Amount      int64
	Status      PaymentStatus
	ProviderRef string // gateway charge id; empty until AUTHORIZED
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Version     int
}

func NewPayment(id, orderID, customerID string, amount int64) (*Payment, error) {
	if id == "" || orderID == "" || customerID == "" {
		return nil, ErrInvalidInput
	}
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	now := time.Now().UTC()
	return &Payment{
		ID:         id,
		OrderID:    orderID,
		CustomerID: customerID,
		Amount:     amount,
		Status:     StatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
		Version:    1,
	}, nil
}

// Authorize marks a successful charge. providerRef is the gateway's charge id.
func (p *Payment) Authorize(providerRef string) error {
	if p.Status != StatusPending {
		return ErrInvalidTransition
	}
	if providerRef == "" {
		return ErrInvalidInput
	}
	p.Status = StatusAuthorized
	p.ProviderRef = providerRef
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// Fail marks the charge as declined.
func (p *Payment) Fail() error {
	if p.Status != StatusPending {
		return ErrInvalidTransition
	}
	p.Status = StatusFailed
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// Release reverses an authorized charge (saga compensation).
func (p *Payment) Release() error {
	if p.Status != StatusAuthorized {
		return ErrInvalidTransition
	}
	p.Status = StatusReleased
	p.UpdatedAt = time.Now().UTC()
	return nil
}
