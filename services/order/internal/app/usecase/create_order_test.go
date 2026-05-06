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

func sampleItems() []domain.Item {
	return []domain.Item{{SKU: "A", Quantity: 2, Price: 1000}}
}

func TestCreateOrder_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockOrderRepository(ctrl)
	idgen := mocks.NewMockIDGenerator(ctrl)

	gomock.InOrder(
		idgen.EXPECT().New().Return("order-1", nil),
		idgen.EXPECT().New().Return("ev-created", nil),
		idgen.EXPECT().New().Return("ev-authorize", nil),
	)

	repo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, o *domain.Order, evs []domain.Event) error {
			assert.Equal(t, "order-1", o.ID)
			assert.Equal(t, "c1", o.CustomerID)
			assert.Equal(t, int64(2000), o.Total)
			assert.Equal(t, domain.StatusPending, o.Status)

			require.Len(t, evs, 2)
			assert.Equal(t, domain.EvtOrderCreated, evs[0].Type)
			assert.Equal(t, "ev-created", evs[0].ID)
			created, ok := evs[0].Payload.(events.OrderCreatedPayload)
			require.True(t, ok)
			assert.Equal(t, "order-1", created.OrderID)
			assert.Equal(t, int64(2000), created.Total)
			assert.Len(t, created.Items, 1)

			assert.Equal(t, domain.CmdAuthorizePayment, evs[1].Type)
			assert.Equal(t, "ev-authorize", evs[1].ID)
			authz, ok := evs[1].Payload.(events.AuthorizePaymentPayload)
			require.True(t, ok)
			assert.Equal(t, "order-1", authz.OrderID)
			assert.Equal(t, int64(2000), authz.Amount)
			return nil
		})

	uc := usecase.NewCreateOrder(repo, idgen, zerolog.Nop())
	order, err := uc.Execute(context.Background(), usecase.CreateOrderInput{
		CustomerID: "c1",
		Items:      sampleItems(),
	})
	require.NoError(t, err)
	assert.Equal(t, "order-1", order.ID)
}

func TestCreateOrder_ValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockOrderRepository(ctrl)
	idgen := mocks.NewMockIDGenerator(ctrl)

	idgen.EXPECT().New().Return("order-1", nil) // first id is generated before validation

	uc := usecase.NewCreateOrder(repo, idgen, zerolog.Nop())
	_, err := uc.Execute(context.Background(), usecase.CreateOrderInput{
		CustomerID: "c1",
		Items:      nil,
	})
	assert.ErrorIs(t, err, domain.ErrEmptyItems)
}

func TestCreateOrder_IDGenErrorPropagates(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockOrderRepository(ctrl)
	idgen := mocks.NewMockIDGenerator(ctrl)

	boom := errors.New("idgen failure")
	idgen.EXPECT().New().Return("", boom)

	uc := usecase.NewCreateOrder(repo, idgen, zerolog.Nop())
	_, err := uc.Execute(context.Background(), usecase.CreateOrderInput{
		CustomerID: "c1",
		Items:      sampleItems(),
	})
	assert.ErrorIs(t, err, boom)
}

func TestCreateOrder_RepoErrorPropagates(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockOrderRepository(ctrl)
	idgen := mocks.NewMockIDGenerator(ctrl)

	idgen.EXPECT().New().Return("order-1", nil)
	idgen.EXPECT().New().Return("ev-1", nil)
	idgen.EXPECT().New().Return("ev-2", nil)

	boom := errors.New("db down")
	repo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(boom)

	uc := usecase.NewCreateOrder(repo, idgen, zerolog.Nop())
	_, err := uc.Execute(context.Background(), usecase.CreateOrderInput{
		CustomerID: "c1",
		Items:      sampleItems(),
	})
	assert.ErrorIs(t, err, boom)
}
