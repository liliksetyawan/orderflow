package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/lilik-setyawan/orderflow/pkg/events"
	"github.com/lilik-setyawan/orderflow/services/payment/internal/app/port"
	"github.com/lilik-setyawan/orderflow/services/payment/internal/app/port/mocks"
	"github.com/lilik-setyawan/orderflow/services/payment/internal/app/usecase"
	"github.com/lilik-setyawan/orderflow/services/payment/internal/domain"
)

func newAuthorizeUC(t *testing.T) (*usecase.AuthorizePayment, *mocks.MockPaymentRepository, *mocks.MockIDGenerator, *mocks.MockPaymentGateway) {
	t.Helper()
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockPaymentRepository(ctrl)
	idgen := mocks.NewMockIDGenerator(ctrl)
	gw := mocks.NewMockPaymentGateway(ctrl)
	uc := usecase.NewAuthorizePayment(repo, idgen, gw, zerolog.Nop())
	return uc, repo, idgen, gw
}

func TestAuthorize_HappyPath(t *testing.T) {
	uc, repo, idgen, gw := newAuthorizeUC(t)

	repo.EXPECT().GetByOrderID(gomock.Any(), "o1").Return(nil, domain.ErrNotFound)
	idgen.EXPECT().New().Return("p1", nil)
	gw.EXPECT().Charge(gomock.Any(), "o1", "c1", int64(1500)).Return("ch_abc", nil)
	idgen.EXPECT().New().Return("ev-1", nil)

	repo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p *domain.Payment, evs []domain.Event) error {
			assert.Equal(t, "p1", p.ID)
			assert.Equal(t, domain.StatusAuthorized, p.Status)
			assert.Equal(t, "ch_abc", p.ProviderRef)

			require.Len(t, evs, 1)
			assert.Equal(t, domain.EvtPaymentAuthorized, evs[0].Type)
			payload, ok := evs[0].Payload.(events.PaymentAuthorizedPayload)
			require.True(t, ok)
			assert.Equal(t, "o1", payload.OrderID)
			assert.Equal(t, "p1", payload.PaymentID)
			return nil
		})

	err := uc.Execute(context.Background(), usecase.AuthorizePaymentInput{
		OrderID:    "o1",
		CustomerID: "c1",
		Amount:     1500,
	})
	require.NoError(t, err)
}

func TestAuthorize_IdempotentSkipWhenExisting(t *testing.T) {
	uc, repo, _, _ := newAuthorizeUC(t)

	existing := &domain.Payment{ID: "p1", OrderID: "o1", Status: domain.StatusAuthorized}
	repo.EXPECT().GetByOrderID(gomock.Any(), "o1").Return(existing, nil)
	// no idgen, no gateway, no Create

	err := uc.Execute(context.Background(), usecase.AuthorizePaymentInput{
		OrderID:    "o1",
		CustomerID: "c1",
		Amount:     1500,
	})
	require.NoError(t, err)
}

func TestAuthorize_DeclinedPublishesFailedEvent(t *testing.T) {
	uc, repo, idgen, gw := newAuthorizeUC(t)

	repo.EXPECT().GetByOrderID(gomock.Any(), "o1").Return(nil, domain.ErrNotFound)
	idgen.EXPECT().New().Return("p1", nil)
	declined := fmt.Errorf("%w: insufficient funds", port.ErrDeclined)
	gw.EXPECT().Charge(gomock.Any(), "o1", "c1", int64(1500)).Return("", declined)
	idgen.EXPECT().New().Return("ev-1", nil)

	repo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p *domain.Payment, evs []domain.Event) error {
			assert.Equal(t, domain.StatusFailed, p.Status)
			assert.Empty(t, p.ProviderRef, "no charge id on decline")

			require.Len(t, evs, 1)
			assert.Equal(t, domain.EvtPaymentFailed, evs[0].Type)
			payload, ok := evs[0].Payload.(events.PaymentFailedPayload)
			require.True(t, ok)
			assert.Equal(t, "o1", payload.OrderID)
			assert.Contains(t, payload.Reason, "insufficient funds")
			return nil
		})

	err := uc.Execute(context.Background(), usecase.AuthorizePaymentInput{
		OrderID:    "o1",
		CustomerID: "c1",
		Amount:     1500,
	})
	// decline is a *successful* business outcome — no error returned to consumer
	require.NoError(t, err)
}

func TestAuthorize_TransientGatewayErrorPropagatesAndDoesNotPersist(t *testing.T) {
	uc, repo, idgen, gw := newAuthorizeUC(t)

	repo.EXPECT().GetByOrderID(gomock.Any(), "o1").Return(nil, domain.ErrNotFound)
	idgen.EXPECT().New().Return("p1", nil)
	transient := errors.New("connection reset")
	gw.EXPECT().Charge(gomock.Any(), "o1", "c1", int64(1500)).Return("", transient)
	// no Create — transient error must NOT persist

	err := uc.Execute(context.Background(), usecase.AuthorizePaymentInput{
		OrderID:    "o1",
		CustomerID: "c1",
		Amount:     1500,
	})
	assert.ErrorIs(t, err, transient)
}

func TestAuthorize_RaceWithAlreadyExistsTreatsAsSuccess(t *testing.T) {
	uc, repo, idgen, gw := newAuthorizeUC(t)

	repo.EXPECT().GetByOrderID(gomock.Any(), "o1").Return(nil, domain.ErrNotFound)
	idgen.EXPECT().New().Return("p1", nil)
	gw.EXPECT().Charge(gomock.Any(), "o1", "c1", int64(1500)).Return("ch_abc", nil)
	idgen.EXPECT().New().Return("ev-1", nil)
	repo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.ErrAlreadyExists)

	err := uc.Execute(context.Background(), usecase.AuthorizePaymentInput{
		OrderID:    "o1",
		CustomerID: "c1",
		Amount:     1500,
	})
	require.NoError(t, err, "race won by parallel instance is OK")
}

func TestAuthorize_PropagatesGetByOrderIDError(t *testing.T) {
	uc, repo, _, _ := newAuthorizeUC(t)

	boom := errors.New("db down")
	repo.EXPECT().GetByOrderID(gomock.Any(), "o1").Return(nil, boom)

	err := uc.Execute(context.Background(), usecase.AuthorizePaymentInput{
		OrderID:    "o1",
		CustomerID: "c1",
		Amount:     1500,
	})
	assert.ErrorIs(t, err, boom)
}
