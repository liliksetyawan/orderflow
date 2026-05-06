package usecase

import (
	"context"

	"github.com/lilik-setyawan/orderflow/services/order/internal/app/port"
	"github.com/lilik-setyawan/orderflow/services/order/internal/domain"
)

type GetOrder struct {
	repo port.OrderRepository
}

func NewGetOrder(repo port.OrderRepository) *GetOrder { return &GetOrder{repo: repo} }

func (uc *GetOrder) Execute(ctx context.Context, id string) (*domain.Order, error) {
	return uc.repo.Get(ctx, id)
}
