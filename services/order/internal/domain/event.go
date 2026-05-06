package domain

import "time"

// EventType is a domain-meaningful name. Adapters translate it into transport
// concerns (e.g. an AMQP routing key) — the domain itself stays unaware.
type EventType string

const (
	// Lifecycle events emitted as the order progresses.
	EvtOrderCreated   EventType = "OrderCreated"
	EvtOrderConfirmed EventType = "OrderConfirmed"
	EvtOrderCanceled  EventType = "OrderCanceled"

	// Saga commands the order service initiates against other services.
	CmdAuthorizePayment EventType = "AuthorizePayment"
	CmdReserveInventory EventType = "ReserveInventory"
	CmdReleasePayment   EventType = "ReleasePayment"
)

// Event is the application-level message produced as a side effect of a use
// case. Payload is opaque here — the adapter that persists it knows how to
// serialize it for the wire.
type Event struct {
	ID         string
	Type       EventType
	Payload    any
	OccurredAt time.Time
}

func NewEvent(id string, t EventType, payload any) Event {
	return Event{
		ID:         id,
		Type:       t,
		Payload:    payload,
		OccurredAt: time.Now().UTC(),
	}
}
