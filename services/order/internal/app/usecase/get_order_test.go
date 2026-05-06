package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/liliksetyawan/orderflow/services/order/internal/app/port/mocks"
	"github.com/liliksetyawan/orderflow/services/order/internal/app/usecase"
	"github.com/liliksetyawan/orderflow/services/order/internal/domain"
)

func TestGetOrder_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockOrderRepository(ctrl)

	want := &domain.Order{ID: "o1", Status: domain.StatusConfirmed}
	repo.EXPECT().Get(gomock.Any(), "o1").Return(want, nil)

	uc := usecase.NewGetOrder(repo)
	got, err := uc.Execute(context.Background(), "o1")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestGetOrder_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockOrderRepository(ctrl)

	repo.EXPECT().Get(gomock.Any(), "o1").Return(nil, domain.ErrNotFound)

	uc := usecase.NewGetOrder(repo)
	_, err := uc.Execute(context.Background(), "o1")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
