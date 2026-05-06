package domain

import "time"

type EventType string

const (
	EvtPaymentAuthorized EventType = "PaymentAuthorized"
	EvtPaymentFailed     EventType = "PaymentFailed"
	EvtPaymentReleased   EventType = "PaymentReleased"
)

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
