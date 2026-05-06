package postgres

import (
	"fmt"

	"github.com/lilik-setyawan/orderflow/pkg/events"
	"github.com/lilik-setyawan/orderflow/services/inventory/internal/domain"
)

type outboxRecord struct {
	ID         string
	RoutingKey string
	Body       []byte
}

func toOutboxRecord(ev domain.Event) (outboxRecord, error) {
	rk, err := routingKey(ev.Type)
	if err != nil {
		return outboxRecord{}, err
	}
	env, err := events.New(ev.ID, rk, ev.Payload)
	if err != nil {
		return outboxRecord{}, err
	}
	env.OccurredAt = ev.OccurredAt
	body, err := env.Marshal()
	if err != nil {
		return outboxRecord{}, err
	}
	return outboxRecord{ID: ev.ID, RoutingKey: rk, Body: body}, nil
}

func routingKey(t domain.EventType) (string, error) {
	switch t {
	case domain.EvtInventoryReserved:
		return "inventory.reserved.v1", nil
	case domain.EvtInventoryFailed:
		return "inventory.failed.v1", nil
	case domain.EvtInventoryReleased:
		return "inventory.released.v1", nil
	default:
		return "", fmt.Errorf("postgres adapter: no routing key for event type %q", t)
	}
}
