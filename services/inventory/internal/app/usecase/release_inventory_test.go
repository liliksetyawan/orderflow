package usecase_test

import (
	"context"
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

func newReleaseUC(t *testing.T) (*usecase.ReleaseInventory, *mocks.MockReservationRepository, *mocks.MockIDGenerator) {
	t.Helper()
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockReservationRepository(ctrl)
	idgen := mocks.NewMockIDGenerator(ctrl)
	uc := usecase.NewReleaseInventory(repo, idgen, zerolog.Nop())
	return uc, repo, idgen
}

func reserved(t *testing.T) *domain.Reservation {
	t.Helper()
	r, err := domain.New("r1", "o1", []domain.Item{{SKU: "A", Quantity: 2}})
	require.NoError(t, err)
	require.NoError(t, r.MarkReserved())
	return r
}

func TestRelease_NoReservationIsNoOp(t *testing.T) {
	uc, repo, _ := newReleaseUC(t)
	repo.EXPECT().GetByOrderID(gomock.Any(), "o1").Return(nil, domain.ErrNotFound)

	err := uc.Execute(context.Background(), usecase.ReleaseInventoryInput{OrderID: "o1"})
	require.NoError(t, err)
}

func TestRelease_AlreadyReleasedIsNoOp(t *testing.T) {
	uc, repo, _ := newReleaseUC(t)
	r := reserved(t)
	require.NoError(t, r.MarkReleased())
	repo.EXPECT().GetByOrderID(gomock.Any(), "o1").Return(r, nil)

	err := uc.Execute(context.Background(), usecase.ReleaseInventoryInput{OrderID: "o1"})
	require.NoError(t, err)
}

func TestRelease_FailedReservationIsNoOp(t *testing.T) {
	uc, repo, _ := newReleaseUC(t)
	r, err := domain.New("r1", "o1", []domain.Item{{SKU: "A", Quantity: 1}})
	require.NoError(t, err)
	require.NoError(t, r.MarkFailed("oos"))
	repo.EXPECT().GetByOrderID(gomock.Any(), "o1").Return(r, nil)

	err = uc.Execute(context.Background(), usecase.ReleaseInventoryInput{OrderID: "o1"})
	require.NoError(t, err)
}

func TestRelease_HappyPath(t *testing.T) {
	uc, repo, idgen := newReleaseUC(t)
	r := reserved(t)
	repo.EXPECT().GetByOrderID(gomock.Any(), "o1").Return(r, nil)
	idgen.EXPECT().New().Return("ev-released", nil)

	repo.EXPECT().
		ReleaseAtomic(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, got *domain.Reservation, evs []domain.Event) error {
			assert.Equal(t, domain.StatusReleased, got.Status)
			require.Len(t, evs, 1)
			assert.Equal(t, domain.EvtInventoryReleased, evs[0].Type)
			payload, ok := evs[0].Payload.(events.InventoryReleasedPayload)
			require.True(t, ok)
			assert.Equal(t, "o1", payload.OrderID)
			return nil
		})

	err := uc.Execute(context.Background(), usecase.ReleaseInventoryInput{OrderID: "o1"})
	require.NoError(t, err)
}
