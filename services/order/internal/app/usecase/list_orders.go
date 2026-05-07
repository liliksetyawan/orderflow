package usecase

import (
	"context"

	"github.com/liliksetyawan/orderflow/services/order/internal/app/port"
	"github.com/liliksetyawan/orderflow/services/order/internal/domain"
)

type ListOrders struct {
	repo port.OrderRepository
}

func NewListOrders(repo port.OrderRepository) *ListOrders { return &ListOrders{repo: repo} }

type ListOrdersInput struct {
	Status string // empty = no filter
	Limit  int
	Offset int
}

type ListOrdersOutput struct {
	Orders []*domain.Order
	Total  int
	Limit  int
	Offset int
}

func (uc *ListOrders) Execute(ctx context.Context, in ListOrdersInput) (*ListOrdersOutput, error) {
	if in.Limit <= 0 || in.Limit > 200 {
		in.Limit = 20
	}
	if in.Offset < 0 {
		in.Offset = 0
	}
	orders, total, err := uc.repo.List(ctx, in.Status, in.Limit, in.Offset)
	if err != nil {
		return nil, err
	}
	return &ListOrdersOutput{
		Orders: orders,
		Total:  total,
		Limit:  in.Limit,
		Offset: in.Offset,
	}, nil
}
