package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lilik-setyawan/orderflow/services/inventory/internal/domain"
)

func TestRoutingKey_AllKnownTypes(t *testing.T) {
	cases := map[domain.EventType]string{
		domain.EvtInventoryReserved: "inventory.reserved.v1",
		domain.EvtInventoryFailed:   "inventory.failed.v1",
		domain.EvtInventoryReleased: "inventory.released.v1",
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
	_, err := routingKey(domain.EventType("UnknownType"))
	assert.Error(t, err)
}
