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
	"github.com/lilik-setyawan/orderflow/services/order/internal/app/port/mocks"
	"github.com/lilik-setyawan/orderflow/services/order/internal/app/usecase"
	"github.com/lilik-setyawan/orderflow/services/order/internal/domain"
)

// orderInStatus builds a fresh Order and walks it to the requested status.
func orderInStatus(t *testing.T, status domain.OrderStatus) *domain.Order {
	t.Helper()
	o, err := domain.NewOrder("o1", "c1", []domain.Item{{SKU: "A", Quantity: 1, Price: 1000}})
	require.NoError(t, err)
	switch status {
	case domain.StatusPending:
	case domain.StatusAuthorized:
		require.NoError(t, o.MarkAuthorized())
	case domain.StatusConfirmed:
		require.NoError(t, o.MarkAuthorized())
		require.NoError(t, o.MarkConfirmed())
	case domain.StatusCanceled:
		require.NoError(t, o.Cancel())
	default:
		t.Fatalf("unsupported status %s", status)
	}
	return o
}

// --- OnPaymentAuthorized ---

func TestOnPaymentAuthorized_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockOrderRepository(ctrl)
	idgen := mocks.NewMockIDGenerator(ctrl)

	o := orderInStatus(t, domain.StatusPending)
	repo.EXPECT().Get(gomock.Any(), "o1").Return(o, nil)
	idgen.EXPECT().New().Return("ev-reserve", nil)

	repo.EXPECT().
		Save(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, got *domain.Order, evs []domain.Event) error {
			assert.Equal(t, domain.StatusAuthorized, got.Status)
			require.Len(t, evs, 1)
			assert.Equal(t, domain.CmdReserveInventory, evs[0].Type)
			payload, ok := evs[0].Payload.(events.ReserveInventoryPayload)
			require.True(t, ok)
			assert.Equal(t, "o1", payload.OrderID)
			assert.Len(t, payload.Items, 1)
			return nil
		})

	saga := usecase.NewSaga(repo, idgen, zerolog.Nop())
	err := saga.OnPaymentAuthorized(context.Background(), events.PaymentAuthorizedPayload{
		OrderID:   "o1",
		PaymentID: "p1",
	})
	require.NoError(t, err)
}

func TestOnPaymentAuthorized_SkipsWhenNotPending(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockOrderRepository(ctrl)
	idgen := mocks.NewMockIDGenerator(ctrl)

	o := orderInStatus(t, domain.StatusAuthorized)
	repo.EXPECT().Get(gomock.Any(), "o1").Return(o, nil)
	// no Save expected; gomock will fail if it gets called

	saga := usecase.NewSaga(repo, idgen, zerolog.Nop())
	err := saga.OnPaymentAuthorized(context.Background(), events.PaymentAuthorizedPayload{
		OrderID: "o1",
	})
	require.NoError(t, err)
	assert.Equal(t, domain.StatusAuthorized, o.Status, "status must not regress")
}

func TestOnPaymentAuthorized_PropagatesGetError(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockOrderRepository(ctrl)
	idgen := mocks.NewMockIDGenerator(ctrl)

	boom := errors.New("db down")
	repo.EXPECT().Get(gomock.Any(), "o1").Return(nil, boom)

	saga := usecase.NewSaga(repo, idgen, zerolog.Nop())
	err := saga.OnPaymentAuthorized(context.Background(), events.PaymentAuthorizedPayload{
		OrderID: "o1",
	})
	assert.ErrorIs(t, err, boom)
}

// --- OnPaymentFailed ---

func TestOnPaymentFailed_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockOrderRepository(ctrl)
	idgen := mocks.NewMockIDGenerator(ctrl)

	o := orderInStatus(t, domain.StatusPending)
	repo.EXPECT().Get(gomock.Any(), "o1").Return(o, nil)
	idgen.EXPECT().New().Return("ev-canceled", nil)

	repo.EXPECT().
		Save(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, got *domain.Order, evs []domain.Event) error {
			assert.Equal(t, domain.StatusCanceled, got.Status)
			require.Len(t, evs, 1)
			assert.Equal(t, domain.EvtOrderCanceled, evs[0].Type)
			payload, ok := evs[0].Payload.(events.OrderCanceledPayload)
			require.True(t, ok)
			assert.Equal(t, "o1", payload.OrderID)
			assert.Equal(t, "c1", payload.CustomerID)
			assert.Contains(t, payload.Reason, "payment_failed:")
			return nil
		})

	saga := usecase.NewSaga(repo, idgen, zerolog.Nop())
	err := saga.OnPaymentFailed(context.Background(), events.PaymentFailedPayload{
		OrderID: "o1",
		Reason:  "card declined",
	})
	require.NoError(t, err)
}

func TestOnPaymentFailed_SkipsWhenNotPending(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockOrderRepository(ctrl)
	idgen := mocks.NewMockIDGenerator(ctrl)

	o := orderInStatus(t, domain.StatusCanceled)
	repo.EXPECT().Get(gomock.Any(), "o1").Return(o, nil)

	saga := usecase.NewSaga(repo, idgen, zerolog.Nop())
	err := saga.OnPaymentFailed(context.Background(), events.PaymentFailedPayload{
		OrderID: "o1",
	})
	require.NoError(t, err)
}

// --- OnInventoryReserved ---

func TestOnInventoryReserved_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockOrderRepository(ctrl)
	idgen := mocks.NewMockIDGenerator(ctrl)

	o := orderInStatus(t, domain.StatusAuthorized)
	repo.EXPECT().Get(gomock.Any(), "o1").Return(o, nil)
	idgen.EXPECT().New().Return("ev-confirmed", nil)

	repo.EXPECT().
		Save(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, got *domain.Order, evs []domain.Event) error {
			assert.Equal(t, domain.StatusConfirmed, got.Status)
			require.Len(t, evs, 1)
			assert.Equal(t, domain.EvtOrderConfirmed, evs[0].Type)
			return nil
		})

	saga := usecase.NewSaga(repo, idgen, zerolog.Nop())
	err := saga.OnInventoryReserved(context.Background(), events.InventoryReservedPayload{
		OrderID: "o1",
	})
	require.NoError(t, err)
}

func TestOnInventoryReserved_SkipsWhenNotAuthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockOrderRepository(ctrl)
	idgen := mocks.NewMockIDGenerator(ctrl)

	o := orderInStatus(t, domain.StatusConfirmed)
	repo.EXPECT().Get(gomock.Any(), "o1").Return(o, nil)

	saga := usecase.NewSaga(repo, idgen, zerolog.Nop())
	err := saga.OnInventoryReserved(context.Background(), events.InventoryReservedPayload{
		OrderID: "o1",
	})
	require.NoError(t, err)
}

// --- OnInventoryFailed ---

func TestOnInventoryFailed_HappyPathEmitsCompensationAndCancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockOrderRepository(ctrl)
	idgen := mocks.NewMockIDGenerator(ctrl)

	o := orderInStatus(t, domain.StatusAuthorized)
	repo.EXPECT().Get(gomock.Any(), "o1").Return(o, nil)
	idgen.EXPECT().New().Return("ev-release", nil)
	idgen.EXPECT().New().Return("ev-canceled", nil)

	repo.EXPECT().
		Save(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, got *domain.Order, evs []domain.Event) error {
			assert.Equal(t, domain.StatusCanceled, got.Status)
			require.Len(t, evs, 2, "must emit ReleasePayment + OrderCanceled atomically")
			assert.Equal(t, domain.CmdReleasePayment, evs[0].Type)
			assert.Equal(t, domain.EvtOrderCanceled, evs[1].Type)

			rel, ok := evs[0].Payload.(events.ReleasePaymentPayload)
			require.True(t, ok)
			assert.Equal(t, "o1", rel.OrderID)
			assert.Equal(t, int64(1000), rel.Amount)

			can, ok := evs[1].Payload.(events.OrderCanceledPayload)
			require.True(t, ok)
			assert.Equal(t, "c1", can.CustomerID)
			assert.Contains(t, can.Reason, "inventory_failed:")
			return nil
		})

	saga := usecase.NewSaga(repo, idgen, zerolog.Nop())
	err := saga.OnInventoryFailed(context.Background(), events.InventoryFailedPayload{
		OrderID: "o1",
		Reason:  "out of stock",
	})
	require.NoError(t, err)
}

func TestOnInventoryFailed_SkipsWhenNotAuthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockOrderRepository(ctrl)
	idgen := mocks.NewMockIDGenerator(ctrl)

	o := orderInStatus(t, domain.StatusPending)
	repo.EXPECT().Get(gomock.Any(), "o1").Return(o, nil)

	saga := usecase.NewSaga(repo, idgen, zerolog.Nop())
	err := saga.OnInventoryFailed(context.Background(), events.InventoryFailedPayload{
		OrderID: "o1",
	})
	require.NoError(t, err)
}

func TestOnInventoryFailed_PropagatesSaveError(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockOrderRepository(ctrl)
	idgen := mocks.NewMockIDGenerator(ctrl)

	o := orderInStatus(t, domain.StatusAuthorized)
	repo.EXPECT().Get(gomock.Any(), "o1").Return(o, nil)
	idgen.EXPECT().New().Return("ev-release", nil)
	idgen.EXPECT().New().Return("ev-canceled", nil)

	boom := errors.New("commit failed")
	repo.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any()).Return(boom)

	saga := usecase.NewSaga(repo, idgen, zerolog.Nop())
	err := saga.OnInventoryFailed(context.Background(), events.InventoryFailedPayload{
		OrderID: "o1",
		Reason:  "stock",
	})
	assert.ErrorIs(t, err, boom)
}
