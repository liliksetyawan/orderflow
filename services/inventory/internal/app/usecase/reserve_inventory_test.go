package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/lilik-setyawan/orderflow/pkg/events"
	"github.com/lilik-setyawan/orderflow/services/inventory/internal/app/port/mocks"
	"github.com/lilik-setyawan/orderflow/services/inventory/internal/app/usecase"
	"github.com/lilik-setyawan/orderflow/services/inventory/internal/domain"
)

func newReserveUC(t *testing.T) (*usecase.ReserveInventory, *mocks.MockReservationRepository, *mocks.MockIDGenerator) {
	t.Helper()
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockReservationRepository(ctrl)
	idgen := mocks.NewMockIDGenerator(ctrl)
	uc := usecase.NewReserveInventory(repo, idgen, zerolog.Nop())
	return uc, repo, idgen
}

func sampleItems() []domain.Item {
	return []domain.Item{{SKU: "A", Quantity: 2}, {SKU: "B", Quantity: 1}}
}

func TestReserve_HappyPath(t *testing.T) {
	uc, repo, idgen := newReserveUC(t)

	repo.EXPECT().GetByOrderID(gomock.Any(), "o1").Return(nil, domain.ErrNotFound)
	idgen.EXPECT().New().Return("r1", nil)
	idgen.EXPECT().New().Return("ev-reserved", nil)

	repo.EXPECT().
		ReserveAtomic(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, r *domain.Reservation, evs []domain.Event) error {
			assert.Equal(t, "r1", r.ID)
			assert.Equal(t, "o1", r.OrderID)
			assert.Equal(t, domain.StatusReserved, r.Status)
			assert.Len(t, r.Items, 2)

			require.Len(t, evs, 1)
			assert.Equal(t, domain.EvtInventoryReserved, evs[0].Type)
			payload, ok := evs[0].Payload.(events.InventoryReservedPayload)
			require.True(t, ok)
			assert.Equal(t, "o1", payload.OrderID)
			assert.Equal(t, "r1", payload.ReservationID)
			return nil
		})

	err := uc.Execute(context.Background(), usecase.ReserveInventoryInput{
		OrderID: "o1",
		Items:   sampleItems(),
	})
	require.NoError(t, err)
}

func TestReserve_IdempotentSkipWhenExisting(t *testing.T) {
	uc, repo, _ := newReserveUC(t)

	existing := &domain.Reservation{ID: "r1", OrderID: "o1", Status: domain.StatusReserved}
	repo.EXPECT().GetByOrderID(gomock.Any(), "o1").Return(existing, nil)

	err := uc.Execute(context.Background(), usecase.ReserveInventoryInput{
		OrderID: "o1",
		Items:   sampleItems(),
	})
	require.NoError(t, err)
}

func TestReserve_InsufficientStockFallsBackToFailed(t *testing.T) {
	uc, repo, idgen := newReserveUC(t)

	repo.EXPECT().GetByOrderID(gomock.Any(), "o1").Return(nil, domain.ErrNotFound)
	idgen.EXPECT().New().Return("r1", nil)
	idgen.EXPECT().New().Return("ev-reserved-attempt", nil)
	repo.EXPECT().ReserveAtomic(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(domain.ErrInsufficientStock)

	idgen.EXPECT().New().Return("r2-failed", nil)
	idgen.EXPECT().New().Return("ev-failed", nil)

	repo.EXPECT().
		RecordFailed(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, r *domain.Reservation, evs []domain.Event) error {
			assert.Equal(t, domain.StatusFailed, r.Status)
			assert.Equal(t, "insufficient stock", r.Reason)

			require.Len(t, evs, 1)
			assert.Equal(t, domain.EvtInventoryFailed, evs[0].Type)
			payload, ok := evs[0].Payload.(events.InventoryFailedPayload)
			require.True(t, ok)
			assert.Equal(t, "o1", payload.OrderID)
			assert.Equal(t, "insufficient stock", payload.Reason)
			return nil
		})

	err := uc.Execute(context.Background(), usecase.ReserveInventoryInput{
		OrderID: "o1",
		Items:   sampleItems(),
	})
	require.NoError(t, err, "insufficient stock is a successful business outcome")
}

func TestReserve_RaceWithAlreadyExistsTreatsAsSuccess(t *testing.T) {
	uc, repo, idgen := newReserveUC(t)

	repo.EXPECT().GetByOrderID(gomock.Any(), "o1").Return(nil, domain.ErrNotFound)
	idgen.EXPECT().New().Return("r1", nil)
	idgen.EXPECT().New().Return("ev-1", nil)
	repo.EXPECT().ReserveAtomic(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(domain.ErrAlreadyExists)

	err := uc.Execute(context.Background(), usecase.ReserveInventoryInput{
		OrderID: "o1",
		Items:   sampleItems(),
	})
	require.NoError(t, err)
}

func TestReserve_TransientReserveErrorPropagates(t *testing.T) {
	uc, repo, idgen := newReserveUC(t)

	repo.EXPECT().GetByOrderID(gomock.Any(), "o1").Return(nil, domain.ErrNotFound)
	idgen.EXPECT().New().Return("r1", nil)
	idgen.EXPECT().New().Return("ev-1", nil)
	boom := errors.New("db down")
	repo.EXPECT().ReserveAtomic(gomock.Any(), gomock.Any(), gomock.Any()).Return(boom)

	err := uc.Execute(context.Background(), usecase.ReserveInventoryInput{
		OrderID: "o1",
		Items:   sampleItems(),
	})
	assert.ErrorIs(t, err, boom)
}
