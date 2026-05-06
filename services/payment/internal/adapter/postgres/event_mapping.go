package postgres

import (
	"fmt"

	"github.com/lilik-setyawan/orderflow/pkg/events"
	"github.com/lilik-setyawan/orderflow/services/payment/internal/domain"
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
	case domain.EvtPaymentAuthorized:
		return "payment.authorized.v1", nil
	case domain.EvtPaymentFailed:
		return "payment.failed.v1", nil
	case domain.EvtPaymentReleased:
		return "payment.released.v1", nil
	default:
		return "", fmt.Errorf("postgres adapter: no routing key for event type %q", t)
	}
}
