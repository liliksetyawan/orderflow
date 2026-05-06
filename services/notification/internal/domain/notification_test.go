package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/liliksetyawan/orderflow/services/notification/internal/domain"
)

func TestNew_HappyPath(t *testing.T) {
	n, err := domain.New("n1", "o1", "c1", domain.TypeOrderConfirmed, "log")
	require.NoError(t, err)
	assert.Equal(t, "n1", n.ID)
	assert.Equal(t, "o1", n.OrderID)
	assert.Equal(t, "c1", n.CustomerID)
	assert.Equal(t, domain.TypeOrderConfirmed, n.Type)
	assert.Equal(t, "log", n.Channel)
	assert.False(t, n.SentAt.IsZero())
}

func TestNew_RejectsEmptyFields(t *testing.T) {
	cases := []struct {
		name                              string
		id, order, customer, typ, channel string
	}{
		{"empty id", "", "o1", "c1", "T", "log"},
		{"empty order", "n1", "", "c1", "T", "log"},
		{"empty customer", "n1", "o1", "", "T", "log"},
		{"empty type", "n1", "o1", "c1", "", "log"},
		{"empty channel", "n1", "o1", "c1", "T", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := domain.New(c.id, c.order, c.customer, c.typ, c.channel)
			assert.ErrorIs(t, err, domain.ErrInvalidInput)
		})
	}
}

func TestTypeConstants(t *testing.T) {
	// Sanity: constants are stable wire values consumed by the postgres
	// UNIQUE(type, ...) idempotency check.
	assert.Equal(t, "OrderConfirmed", domain.TypeOrderConfirmed)
	assert.Equal(t, "OrderCanceled", domain.TypeOrderCanceled)
}
