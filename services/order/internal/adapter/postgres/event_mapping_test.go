package postgres

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lilik-setyawan/orderflow/pkg/events"
	"github.com/lilik-setyawan/orderflow/services/order/internal/domain"
)

func TestRoutingKey_AllKnownTypes(t *testing.T) {
	cases := map[domain.EventType]string{
		domain.EvtOrderCreated:     "order.created.v1",
		domain.EvtOrderConfirmed:   "order.confirmed.v1",
		domain.EvtOrderCanceled:    "order.canceled.v1",
		domain.CmdAuthorizePayment: "payment.authorize.v1",
		domain.CmdReserveInventory: "inventory.reserve.v1",
		domain.CmdReleasePayment:   "payment.release.v1",
	}
	for typ, want := range cases {
		t.Run(string(typ), func(t *testing.T) {
			got, err := routingKey(typ)
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}

func TestRoutingKey_RejectsUnknownType(t *testing.T) {
	_, err := routingKey(domain.EventType("MysteryType"))
	assert.Error(t, err)
}

func TestToOutboxRecord_PreservesIDAndPayload(t *testing.T) {
	ev := domain.NewEvent("ev-1", domain.EvtOrderCreated, events.OrderCreatedPayload{
		OrderID:    "o1",
		CustomerID: "c1",
		Total:      999,
	})
	rec, err := toOutboxRecord(ev)
	require.NoError(t, err)
	assert.Equal(t, "ev-1", rec.ID)
	assert.Equal(t, "order.created.v1", rec.RoutingKey)

	env, err := events.Unmarshal(rec.Body)
	require.NoError(t, err)
	assert.Equal(t, "ev-1", env.ID)
	assert.Equal(t, "order.created.v1", env.Type)

	var got events.OrderCreatedPayload
	require.NoError(t, json.Unmarshal(env.Payload, &got))
	assert.Equal(t, "o1", got.OrderID)
	assert.Equal(t, int64(999), got.Total)
}
