// Package gateway is the driven adapter for port.PaymentGateway. The Mock
// implementation simulates a real gateway with configurable success rate
// and latency — good enough for end-to-end saga demos without dragging
// Stripe/Midtrans/etc into the dev loop.
package gateway

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"

	"github.com/liliksetyawan/orderflow/services/payment/internal/app/port"
)

type Mock struct {
	successRate float64 // 0.0 to 1.0
	minLatency  time.Duration
	maxLatency  time.Duration
}

func NewMock(successRate float64) *Mock {
	if successRate < 0 {
		successRate = 0
	}
	if successRate > 1 {
		successRate = 1
	}
	return &Mock{
		successRate: successRate,
		minLatency:  50 * time.Millisecond,
		maxLatency:  300 * time.Millisecond,
	}
}

var _ port.PaymentGateway = (*Mock)(nil)

func (m *Mock) Charge(ctx context.Context, orderID, customerID string, amount int64) (string, error) {
	if err := simulateLatency(ctx, m.minLatency, m.maxLatency); err != nil {
		return "", err
	}
	if rand.Float64() > m.successRate {
		return "", fmt.Errorf("%w: insufficient funds", port.ErrDeclined)
	}
	return "ch_" + uuid.NewString(), nil
}

func (m *Mock) Refund(ctx context.Context, providerRef string, amount int64) error {
	return simulateLatency(ctx, m.minLatency, m.maxLatency)
}

func simulateLatency(ctx context.Context, low, high time.Duration) error {
	delta := high - low
	d := low
	if delta > 0 {
		d += time.Duration(rand.Int64N(int64(delta)))
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}
