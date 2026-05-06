package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/liliksetyawan/orderflow/services/inventory/internal/domain"
)

func items() []domain.Item {
	return []domain.Item{{SKU: "A", Quantity: 2}}
}

func TestNew_HappyPath(t *testing.T) {
	r, err := domain.New("r1", "o1", items())
	require.NoError(t, err)
	assert.Equal(t, "r1", r.ID)
	assert.Equal(t, "o1", r.OrderID)
	assert.Empty(t, r.Status, "status starts unset; caller must Mark*")
	assert.Equal(t, 1, r.Version)
	assert.Len(t, r.Items, 1)
}

func TestNew_RejectsEmptyFields(t *testing.T) {
	t.Run("empty id", func(t *testing.T) {
		_, err := domain.New("", "o1", items())
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})
	t.Run("empty order", func(t *testing.T) {
		_, err := domain.New("r1", "", items())
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})
}

func TestNew_RejectsEmptyItems(t *testing.T) {
	_, err := domain.New("r1", "o1", nil)
	assert.ErrorIs(t, err, domain.ErrEmptyItems)
}

func TestNew_RejectsBadItem(t *testing.T) {
	cases := map[string][]domain.Item{
		"empty sku":    {{SKU: "", Quantity: 1}},
		"zero qty":     {{SKU: "A", Quantity: 0}},
		"negative qty": {{SKU: "A", Quantity: -3}},
	}
	for name, its := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := domain.New("r1", "o1", its)
			assert.ErrorIs(t, err, domain.ErrInvalidItem)
		})
	}
}

func TestMarkReserved_HappyPath(t *testing.T) {
	r, _ := domain.New("r1", "o1", items())
	require.NoError(t, r.MarkReserved())
	assert.Equal(t, domain.StatusReserved, r.Status)
}

func TestMarkReserved_RejectsNonInitialStatus(t *testing.T) {
	r, _ := domain.New("r1", "o1", items())
	require.NoError(t, r.MarkReserved())
	assert.ErrorIs(t, r.MarkReserved(), domain.ErrInvalidTransition)
}

func TestMarkFailed_HappyPath(t *testing.T) {
	r, _ := domain.New("r1", "o1", items())
	require.NoError(t, r.MarkFailed("oos"))
	assert.Equal(t, domain.StatusFailed, r.Status)
	assert.Equal(t, "oos", r.Reason)
}

func TestMarkFailed_RejectsNonInitialStatus(t *testing.T) {
	r, _ := domain.New("r1", "o1", items())
	require.NoError(t, r.MarkReserved())
	assert.ErrorIs(t, r.MarkFailed("late"), domain.ErrInvalidTransition)
}

func TestMarkReleased_HappyPath(t *testing.T) {
	r, _ := domain.New("r1", "o1", items())
	require.NoError(t, r.MarkReserved())
	require.NoError(t, r.MarkReleased())
	assert.Equal(t, domain.StatusReleased, r.Status)
}

func TestMarkReleased_RejectsNonReservedStatus(t *testing.T) {
	t.Run("from initial", func(t *testing.T) {
		r, _ := domain.New("r1", "o1", items())
		assert.ErrorIs(t, r.MarkReleased(), domain.ErrInvalidTransition)
	})
	t.Run("from failed", func(t *testing.T) {
		r, _ := domain.New("r1", "o1", items())
		require.NoError(t, r.MarkFailed("oos"))
		assert.ErrorIs(t, r.MarkReleased(), domain.ErrInvalidTransition)
	})
	t.Run("from released", func(t *testing.T) {
		r, _ := domain.New("r1", "o1", items())
		require.NoError(t, r.MarkReserved())
		require.NoError(t, r.MarkReleased())
		assert.ErrorIs(t, r.MarkReleased(), domain.ErrInvalidTransition)
	})
}
