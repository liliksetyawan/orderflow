// Package events defines the wire format for asynchronous events on NATS.
//
// Every event is wrapped in an Envelope so consumers can route, dedupe, and
// trace without parsing the inner payload. The envelope carries:
//
//   - ID           globally unique event id (uuid v7) — sent as the AMQP
//     MessageId by the publisher and used by consumers as the
//     idempotency key
//   - Type         dotted name, e.g. "order.created.v1"
//   - OccurredAt   when the event was produced (not when published)
//   - TraceParent  W3C traceparent header — keeps spans linked across the bus
//   - Payload      domain-specific JSON
package events

import (
	"encoding/json"
	"time"
)

type Envelope struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	OccurredAt  time.Time       `json:"occurred_at"`
	TraceParent string          `json:"traceparent,omitempty"`
	Payload     json.RawMessage `json:"payload"`
}

func New(id, typ string, payload any) (Envelope, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, err
	}
	return Envelope{
		ID:         id,
		Type:       typ,
		OccurredAt: time.Now().UTC(),
		Payload:    b,
	}, nil
}

func (e Envelope) Marshal() ([]byte, error) { return json.Marshal(e) }
func Unmarshal(b []byte) (Envelope, error) {
	var e Envelope
	err := json.Unmarshal(b, &e)
	return e, err
}

// Common event type constants. Versioned so we can evolve payloads safely.
const (
	TypeOrderCreated      = "order.created.v1"
	TypeOrderConfirmed    = "order.confirmed.v1"
	TypeOrderCanceled     = "order.canceled.v1"
	TypePaymentAuthorized = "payment.authorized.v1"
	TypePaymentFailed     = "payment.failed.v1"
	TypePaymentReleased   = "payment.released.v1"
	TypeInventoryReserved = "inventory.reserved.v1"
	TypeInventoryFailed   = "inventory.failed.v1"
	TypeInventoryReleased = "inventory.released.v1"
)
