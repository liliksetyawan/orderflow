package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/liliksetyawan/orderflow/pkg/events"
	"github.com/liliksetyawan/orderflow/services/payment/internal/app/port/mocks"
	"github.com/liliksetyawan/orderflow/services/payment/internal/app/usecase"
	"github.com/liliksetyawan/orderflow/services/payment/internal/domain"
)

func newReleaseUC(t *testing.T) (*usecase.ReleasePayment, *mocks.MockPaymentRepository, *mocks.MockIDGenerator, *mocks.MockPaymentGateway) {
	t.Helper()
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockPaymentRepository(ctrl)
	idgen := mocks.NewMockIDGenerator(ctrl)
	gw := mocks.NewMockPaymentGateway(ctrl)
	uc := usecase.NewReleasePayment(repo, idgen, gw, zerolog.Nop())
	return uc, repo, idgen, gw
}

func TestRelease_NoPaymentIsNoOp(t *testing.T) {
	uc, repo, _, _ := newReleaseUC(t)
	repo.EXPECT().GetByOrderID(gomock.Any(), "o1").Return(nil, domain.ErrNotFound)

	err := uc.Execute(context.Background(), usecase.ReleasePaymentInput{OrderID: "o1"})
	require.NoError(t, err)
}

func TestRelease_AlreadyReleasedIsNoOp(t *testing.T) {
	uc, repo, _, _ := newReleaseUC(t)
	repo.EXPECT().GetByOrderID(gomock.Any(), "o1").Return(&domain.Payment{
		ID: "p1", OrderID: "o1", Status: domain.StatusReleased,
	}, nil)
	// no Refund, no Save

	err := uc.Execute(context.Background(), usecase.ReleasePaymentInput{OrderID: "o1"})
	require.NoError(t, err)
}

func TestRelease_FailedPaymentIsNoOp(t *testing.T) {
	uc, repo, _, _ := newReleaseUC(t)
	repo.EXPECT().GetByOrderID(gomock.Any(), "o1").Return(&domain.Payment{
		ID: "p1", OrderID: "o1", Status: domain.StatusFailed,
	}, nil)

	err := uc.Execute(context.Background(), usecase.ReleasePaymentInput{OrderID: "o1"})
	require.NoError(t, err)
}

func TestRelease_PendingPaymentIsNoOp(t *testing.T) {
	uc, repo, _, _ := newReleaseUC(t)
	repo.EXPECT().GetByOrderID(gomock.Any(), "o1").Return(&domain.Payment{
		ID: "p1", OrderID: "o1", Status: domain.StatusPending,
	}, nil)

	err := uc.Execute(context.Background(), usecase.ReleasePaymentInput{OrderID: "o1"})
	require.NoError(t, err)
}

func TestRelease_HappyPath(t *testing.T) {
	uc, repo, idgen, gw := newReleaseUC(t)

	authorized, err := domain.NewPayment("p1", "o1", "c1", 2000)
	require.NoError(t, err)
	require.NoError(t, authorized.Authorize("ch_abc"))

	repo.EXPECT().GetByOrderID(gomock.Any(), "o1").Return(authorized, nil)
	gw.EXPECT().Refund(gomock.Any(), "ch_abc", int64(2000)).Return(nil)
	idgen.EXPECT().New().Return("ev-released", nil)

	repo.EXPECT().
		Save(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p *domain.Payment, evs []domain.Event) error {
			assert.Equal(t, domain.StatusReleased, p.Status)
			require.Len(t, evs, 1)
			assert.Equal(t, domain.EvtPaymentReleased, evs[0].Type)
			payload, ok := evs[0].Payload.(events.PaymentReleasedPayload)
			require.True(t, ok)
			assert.Equal(t, "o1", payload.OrderID)
			return nil
		})

	err = uc.Execute(context.Background(), usecase.ReleasePaymentInput{OrderID: "o1", Amount: 2000})
	require.NoError(t, err)
}

func TestRelease_RefundErrorPropagates(t *testing.T) {
	uc, repo, _, gw := newReleaseUC(t)

	authorized, err := domain.NewPayment("p1", "o1", "c1", 2000)
	require.NoError(t, err)
	require.NoError(t, authorized.Authorize("ch_abc"))

	repo.EXPECT().GetByOrderID(gomock.Any(), "o1").Return(authorized, nil)
	boom := errors.New("network error")
	gw.EXPECT().Refund(gomock.Any(), "ch_abc", int64(2000)).Return(boom)
	// no Save expected — refund failure stops the flow

	err = uc.Execute(context.Background(), usecase.ReleasePaymentInput{OrderID: "o1"})
	assert.ErrorIs(t, err, boom)
}
