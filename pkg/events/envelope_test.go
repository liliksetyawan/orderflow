package events_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/liliksetyawan/orderflow/pkg/events"
)

func TestNew_PopulatesAllFields(t *testing.T) {
	payload := events.OrderCreatedPayload{
		OrderID:    "o1",
		CustomerID: "c1",
		Total:      1000,
	}
	env, err := events.New("evt-1", "order.created.v1", payload)
	require.NoError(t, err)
	assert.Equal(t, "evt-1", env.ID)
	assert.Equal(t, "order.created.v1", env.Type)
	assert.False(t, env.OccurredAt.IsZero())
	assert.NotEmpty(t, env.Payload, "payload should be marshaled JSON")

	var got events.OrderCreatedPayload
	require.NoError(t, json.Unmarshal(env.Payload, &got))
	assert.Equal(t, payload, got)
}

func TestMarshalUnmarshal_Roundtrip(t *testing.T) {
	original, err := events.New("evt-1", "payment.authorized.v1",
		events.PaymentAuthorizedPayload{OrderID: "o1", PaymentID: "p1"})
	require.NoError(t, err)

	wire, err := original.Marshal()
	require.NoError(t, err)

	got, err := events.Unmarshal(wire)
	require.NoError(t, err)
	assert.Equal(t, original.ID, got.ID)
	assert.Equal(t, original.Type, got.Type)
	assert.WithinDuration(t, original.OccurredAt, got.OccurredAt, 0)

	var payload events.PaymentAuthorizedPayload
	require.NoError(t, json.Unmarshal(got.Payload, &payload))
	assert.Equal(t, "o1", payload.OrderID)
	assert.Equal(t, "p1", payload.PaymentID)
}

func TestUnmarshal_RejectsBadJSON(t *testing.T) {
	_, err := events.Unmarshal([]byte("not-json"))
	assert.Error(t, err)
}
