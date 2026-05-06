package domain

import "time"

type EventType string

const (
	EvtInventoryReserved EventType = "InventoryReserved"
	EvtInventoryFailed   EventType = "InventoryFailed"
	EvtInventoryReleased EventType = "InventoryReleased"
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
