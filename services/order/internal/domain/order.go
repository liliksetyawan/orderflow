// Package domain holds the Order aggregate and its business rules. This
// package is the core of the hexagonal layout: it has zero dependencies on
// infrastructure (no pgx, no rabbitmq, no zerolog, no JSON tags) so it can be
// tested in isolation and reused across adapters.
package domain

import "time"

type OrderStatus string

const (
	StatusPending    OrderStatus = "PENDING"    // just created, awaiting payment
	StatusAuthorized OrderStatus = "AUTHORIZED" // payment ok, awaiting inventory
	StatusConfirmed  OrderStatus = "CONFIRMED"  // saga completed successfully
	StatusCanceled   OrderStatus = "CANCELED"   // saga compensated / failed
)

type Order struct {
	ID         string
	CustomerID string
	Items      []Item
	Total      int64
	Status     OrderStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Version    int // optimistic concurrency token
}

type Item struct {
	SKU      string
	Quantity int
	Price    int64 // minor units (e.g. cents)
}

// NewOrder validates and constructs a fresh Order in PENDING status.
// The id is supplied by the caller (use case) so the domain stays free of
// infrastructure concerns like UUID generation.
func NewOrder(id, customerID string, items []Item) (*Order, error) {
	if customerID == "" {
		return nil, ErrInvalidCustomer
	}
	if len(items) == 0 {
		return nil, ErrEmptyItems
	}

	var total int64
	for _, it := range items {
		if it.SKU == "" || it.Quantity <= 0 || it.Price < 0 {
			return nil, ErrInvalidItem
		}
		total += int64(it.Quantity) * it.Price
	}

	now := time.Now().UTC()
	return &Order{
		ID:         id,
		CustomerID: customerID,
		Items:      items,
		Total:      total,
		Status:     StatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
		Version:    1,
	}, nil
}

// MarkAuthorized: PENDING → AUTHORIZED. Called when payment succeeds.
func (o *Order) MarkAuthorized() error {
	if o.Status != StatusPending {
		return ErrInvalidTransition
	}
	o.Status = StatusAuthorized
	o.UpdatedAt = time.Now().UTC()
	return nil
}

// MarkConfirmed: AUTHORIZED → CONFIRMED. Called when inventory reservation succeeds.
func (o *Order) MarkConfirmed() error {
	if o.Status != StatusAuthorized {
		return ErrInvalidTransition
	}
	o.Status = StatusConfirmed
	o.UpdatedAt = time.Now().UTC()
	return nil
}

// Cancel: PENDING or AUTHORIZED → CANCELED. Terminal-state orders cannot be canceled.
func (o *Order) Cancel() error {
	if o.Status == StatusConfirmed || o.Status == StatusCanceled {
		return ErrInvalidTransition
	}
	o.Status = StatusCanceled
	o.UpdatedAt = time.Now().UTC()
	return nil
}
