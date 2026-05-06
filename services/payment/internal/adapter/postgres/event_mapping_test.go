package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/liliksetyawan/orderflow/services/payment/internal/domain"
)

func TestRoutingKey_AllKnownTypes(t *testing.T) {
	cases := map[domain.EventType]string{
		domain.EvtPaymentAuthorized: "payment.authorized.v1",
		domain.EvtPaymentFailed:     "payment.failed.v1",
		domain.EvtPaymentReleased:   "payment.released.v1",
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
	_, err := routingKey(domain.EventType("BogusType"))
	assert.Error(t, err)
}
