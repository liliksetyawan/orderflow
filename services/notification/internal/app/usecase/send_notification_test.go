package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/liliksetyawan/orderflow/services/notification/internal/app/port/mocks"
	"github.com/liliksetyawan/orderflow/services/notification/internal/app/usecase"
	"github.com/liliksetyawan/orderflow/services/notification/internal/domain"
)

func newSendUC(t *testing.T) (*usecase.SendNotification, *mocks.MockNotificationRepository, *mocks.MockNotifier, *mocks.MockIDGenerator) {
	t.Helper()
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockNotificationRepository(ctrl)
	notif := mocks.NewMockNotifier(ctrl)
	idgen := mocks.NewMockIDGenerator(ctrl)
	uc := usecase.NewSendNotification(repo, notif, idgen, zerolog.Nop())
	return uc, repo, notif, idgen
}

func TestSend_HappyPath(t *testing.T) {
	uc, repo, notif, idgen := newSendUC(t)

	notif.EXPECT().Channel().Return("log").AnyTimes()
	repo.EXPECT().GetByOrderTypeChannel(gomock.Any(), "o1", domain.TypeOrderConfirmed, "log").
		Return(nil, domain.ErrNotFound)
	idgen.EXPECT().New().Return("n1", nil)
	notif.EXPECT().
		Send(gomock.Any(), gomock.Any(), "").
		DoAndReturn(func(_ context.Context, n domain.Notification, reason string) error {
			assert.Equal(t, "n1", n.ID)
			assert.Equal(t, "o1", n.OrderID)
			assert.Equal(t, "c1", n.CustomerID)
			assert.Equal(t, domain.TypeOrderConfirmed, n.Type)
			return nil
		})
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	err := uc.Execute(context.Background(), usecase.SendNotificationInput{
		OrderID:    "o1",
		CustomerID: "c1",
		Type:       domain.TypeOrderConfirmed,
	})
	require.NoError(t, err)
}

func TestSend_IdempotentSkipWhenAlreadyRecorded(t *testing.T) {
	uc, repo, notif, _ := newSendUC(t)

	notif.EXPECT().Channel().Return("log").AnyTimes()
	repo.EXPECT().GetByOrderTypeChannel(gomock.Any(), "o1", domain.TypeOrderConfirmed, "log").
		Return(&domain.Notification{ID: "n1"}, nil)
	// no Send, no Create

	err := uc.Execute(context.Background(), usecase.SendNotificationInput{
		OrderID:    "o1",
		CustomerID: "c1",
		Type:       domain.TypeOrderConfirmed,
	})
	require.NoError(t, err)
}

func TestSend_SendFailureReturnsErrorWithoutPersist(t *testing.T) {
	uc, repo, notif, idgen := newSendUC(t)

	notif.EXPECT().Channel().Return("log").AnyTimes()
	repo.EXPECT().GetByOrderTypeChannel(gomock.Any(), "o1", domain.TypeOrderCanceled, "log").
		Return(nil, domain.ErrNotFound)
	idgen.EXPECT().New().Return("n1", nil)
	boom := errors.New("smtp timeout")
	notif.EXPECT().Send(gomock.Any(), gomock.Any(), "user_canceled").Return(boom)
	// no Create — failed sends are not recorded; consumer requeues

	err := uc.Execute(context.Background(), usecase.SendNotificationInput{
		OrderID:    "o1",
		CustomerID: "c1",
		Type:       domain.TypeOrderCanceled,
		Reason:     "user_canceled",
	})
	assert.ErrorIs(t, err, boom)
}

func TestSend_RaceWithAlreadyExistsTreatsAsSuccess(t *testing.T) {
	uc, repo, notif, idgen := newSendUC(t)

	notif.EXPECT().Channel().Return("log").AnyTimes()
	repo.EXPECT().GetByOrderTypeChannel(gomock.Any(), "o1", domain.TypeOrderConfirmed, "log").
		Return(nil, domain.ErrNotFound)
	idgen.EXPECT().New().Return("n1", nil)
	notif.EXPECT().Send(gomock.Any(), gomock.Any(), "").Return(nil)
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(domain.ErrAlreadyExists)

	err := uc.Execute(context.Background(), usecase.SendNotificationInput{
		OrderID:    "o1",
		CustomerID: "c1",
		Type:       domain.TypeOrderConfirmed,
	})
	require.NoError(t, err)
}

func TestSend_PassesReasonForCancel(t *testing.T) {
	uc, repo, notif, idgen := newSendUC(t)

	notif.EXPECT().Channel().Return("log").AnyTimes()
	repo.EXPECT().GetByOrderTypeChannel(gomock.Any(), "o1", domain.TypeOrderCanceled, "log").
		Return(nil, domain.ErrNotFound)
	idgen.EXPECT().New().Return("n1", nil)
	notif.EXPECT().Send(gomock.Any(), gomock.Any(), "payment_failed").Return(nil)
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	err := uc.Execute(context.Background(), usecase.SendNotificationInput{
		OrderID:    "o1",
		CustomerID: "c1",
		Type:       domain.TypeOrderCanceled,
		Reason:     "payment_failed",
	})
	require.NoError(t, err)
}
