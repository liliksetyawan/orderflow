// Package domain holds the Reservation aggregate and its business rules.
// Pure: no pgx, no rabbitmq, no zerolog. State machine is:
//
//	(empty) ─┬─> RESERVED ──> RELEASED
//	         └─> FAILED
package domain

import "time"

type ReservationStatus string

const (
	StatusReserved ReservationStatus = "RESERVED"
	StatusFailed   ReservationStatus = "FAILED"
	StatusReleased ReservationStatus = "RELEASED"
)

type Reservation struct {
	ID         string
	OrderID    string
	Items      []Item
	Status     ReservationStatus
	Reason     string // populated only when FAILED
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Version    int
}

type Item struct {
	SKU      string
	Quantity int
}

func New(id, orderID string, items []Item) (*Reservation, error) {
	if id == "" || orderID == "" {
		return nil, ErrInvalidInput
	}
	if len(items) == 0 {
		return nil, ErrEmptyItems
	}
	for _, it := range items {
		if it.SKU == "" || it.Quantity <= 0 {
			return nil, ErrInvalidItem
		}
	}
	now := time.Now().UTC()
	return &Reservation{
		ID:        id,
		OrderID:   orderID,
		Items:     items,
		CreatedAt: now,
		UpdatedAt: now,
		Version:   1,
	}, nil
}

func (r *Reservation) MarkReserved() error {
	if r.Status != "" {
		return ErrInvalidTransition
	}
	r.Status = StatusReserved
	r.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *Reservation) MarkFailed(reason string) error {
	if r.Status != "" {
		return ErrInvalidTransition
	}
	r.Status = StatusFailed
	r.Reason = reason
	r.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *Reservation) MarkReleased() error {
	if r.Status != StatusReserved {
		return ErrInvalidTransition
	}
	r.Status = StatusReleased
	r.UpdatedAt = time.Now().UTC()
	return nil
}
