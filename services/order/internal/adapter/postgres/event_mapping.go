package postgres

import (
	"fmt"

	"github.com/liliksetyawan/orderflow/pkg/events"
	"github.com/liliksetyawan/orderflow/services/order/internal/domain"
)

// outboxRecord is the row we INSERT into outbox.
type outboxRecord struct {
	ID         string
	RoutingKey string
	Body       []byte
}

// toOutboxRecord serializes a domain.Event to its wire form. It's the only
// place that knows about the AMQP routing key conventions — keeps the domain
// blissfully unaware of transport.
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
	case domain.EvtOrderCreated:
		return "order.created.v1", nil
	case domain.EvtOrderConfirmed:
		return "order.confirmed.v1", nil
	case domain.EvtOrderCanceled:
		return "order.canceled.v1", nil
	case domain.CmdAuthorizePayment:
		return "payment.authorize.v1", nil
	case domain.CmdReserveInventory:
		return "inventory.reserve.v1", nil
	case domain.CmdReleasePayment:
		return "payment.release.v1", nil
	default:
		return "", fmt.Errorf("postgres adapter: no routing key for event type %q", t)
	}
}
