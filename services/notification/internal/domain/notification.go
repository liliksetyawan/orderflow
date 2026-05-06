// Package domain holds the Notification record. The notification service
// only persists *successful* sends — failed sends are returned to the
// consumer for retry and leave no row behind. So there's no status field
// or transition machinery: a Notification's existence already means
// "we sent this".
package domain

import "time"

// Type values used on the wire and in the DB. Kept as constants so the
// consumer dispatch and the DB rows stay in sync.
const (
	TypeOrderConfirmed = "OrderConfirmed"
	TypeOrderCanceled  = "OrderCanceled"
)

type Notification struct {
	ID         string
	OrderID    string
	CustomerID string
	Type       string // see Type* constants above
	Channel    string // "log", "email", "push" — determined by the Notifier adapter
	SentAt     time.Time
}

func New(id, orderID, customerID, typ, channel string) (*Notification, error) {
	if id == "" || orderID == "" || customerID == "" || typ == "" || channel == "" {
		return nil, ErrInvalidInput
	}
	return &Notification{
		ID:         id,
		OrderID:    orderID,
		CustomerID: customerID,
		Type:       typ,
		Channel:    channel,
		SentAt:     time.Now().UTC(),
	}, nil
}
