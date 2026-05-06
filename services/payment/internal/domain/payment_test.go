package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lilik-setyawan/orderflow/services/payment/internal/domain"
)

func TestNewPayment_HappyPath(t *testing.T) {
	p, err := domain.NewPayment("p1", "o1", "c1", 1500)
	require.NoError(t, err)
	assert.Equal(t, "p1", p.ID)
	assert.Equal(t, "o1", p.OrderID)
	assert.Equal(t, int64(1500), p.Amount)
	assert.Equal(t, domain.StatusPending, p.Status)
	assert.Equal(t, 1, p.Version)
	assert.Empty(t, p.ProviderRef)
	assert.False(t, p.CreatedAt.IsZero())
}

func TestNewPayment_RejectsEmptyFields(t *testing.T) {
	cases := []struct {
		name                       string
		id, order, customer string
		amount              int64
	}{
		{"empty id", "", "o1", "c1", 100},
		{"empty order", "p1", "", "c1", 100},
		{"empty customer", "p1", "o1", "", 100},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := domain.NewPayment(c.id, c.order, c.customer, c.amount)
			assert.ErrorIs(t, err, domain.ErrInvalidInput)
		})
	}
}

func TestNewPayment_RejectsNonPositiveAmount(t *testing.T) {
	for _, amount := range []int64{0, -1, -100} {
		_, err := domain.NewPayment("p1", "o1", "c1", amount)
		assert.ErrorIs(t, err, domain.ErrInvalidAmount, "amount=%d", amount)
	}
}

func TestAuthorize_HappyPath(t *testing.T) {
	p, _ := domain.NewPayment("p1", "o1", "c1", 1000)
	require.NoError(t, p.Authorize("ch_abc"))
	assert.Equal(t, domain.StatusAuthorized, p.Status)
	assert.Equal(t, "ch_abc", p.ProviderRef)
}

func TestAuthorize_RejectsEmptyProviderRef(t *testing.T) {
	p, _ := domain.NewPayment("p1", "o1", "c1", 1000)
	assert.ErrorIs(t, p.Authorize(""), domain.ErrInvalidInput)
	assert.Equal(t, domain.StatusPending, p.Status, "status must not change on error")
}

func TestAuthorize_RejectsNonPendingTransitions(t *testing.T) {
	p, _ := domain.NewPayment("p1", "o1", "c1", 1000)
	require.NoError(t, p.Authorize("ch_abc"))
	assert.ErrorIs(t, p.Authorize("ch_def"), domain.ErrInvalidTransition)
}

func TestFail_HappyPath(t *testing.T) {
	p, _ := domain.NewPayment("p1", "o1", "c1", 1000)
	require.NoError(t, p.Fail())
	assert.Equal(t, domain.StatusFailed, p.Status)
}

func TestFail_RejectsNonPendingTransitions(t *testing.T) {
	p, _ := domain.NewPayment("p1", "o1", "c1", 1000)
	require.NoError(t, p.Authorize("ch_abc"))
	assert.ErrorIs(t, p.Fail(), domain.ErrInvalidTransition)
}

func TestRelease_HappyPath(t *testing.T) {
	p, _ := domain.NewPayment("p1", "o1", "c1", 1000)
	require.NoError(t, p.Authorize("ch_abc"))
	require.NoError(t, p.Release())
	assert.Equal(t, domain.StatusReleased, p.Status)
}

func TestRelease_RejectsFromPendingOrFailed(t *testing.T) {
	t.Run("pending", func(t *testing.T) {
		p, _ := domain.NewPayment("p1", "o1", "c1", 1000)
		assert.ErrorIs(t, p.Release(), domain.ErrInvalidTransition)
	})
	t.Run("failed", func(t *testing.T) {
		p, _ := domain.NewPayment("p1", "o1", "c1", 1000)
		require.NoError(t, p.Fail())
		assert.ErrorIs(t, p.Release(), domain.ErrInvalidTransition)
	})
	t.Run("released", func(t *testing.T) {
		p, _ := domain.NewPayment("p1", "o1", "c1", 1000)
		require.NoError(t, p.Authorize("ch"))
		require.NoError(t, p.Release())
		assert.ErrorIs(t, p.Release(), domain.ErrInvalidTransition)
	})
}
