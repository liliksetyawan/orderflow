// Package usecase contains the application services — concrete orchestrations
// that fulfill product use cases by composing the domain over the ports.
package usecase

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/lilik-setyawan/orderflow/pkg/events"
	"github.com/lilik-setyawan/orderflow/services/order/internal/app/port"
	"github.com/lilik-setyawan/orderflow/services/order/internal/domain"
)

type CreateOrder struct {
	repo  port.OrderRepository
	idgen port.IDGenerator
	log   zerolog.Logger
}

func NewCreateOrder(repo port.OrderRepository, idgen port.IDGenerator, log zerolog.Logger) *CreateOrder {
	return &CreateOrder{
		repo:  repo,
		idgen: idgen,
		log:   log.With().Str("usecase", "create_order").Logger(),
	}
}

type CreateOrderInput struct {
	CustomerID string
	Items      []domain.Item
}

// Execute creates an Order in PENDING status and emits two events transactionally:
//   - OrderCreated   — broadcast for downstream subscribers (notification etc.)
//   - AuthorizePayment — saga command directing the payment service to charge
//
// Both events plus the order row commit together via the outbox.
func (uc *CreateOrder) Execute(ctx context.Context, in CreateOrderInput) (*domain.Order, error) {
	orderID, err := uc.idgen.New()
	if err != nil {
		return nil, err
	}

	order, err := domain.NewOrder(orderID, in.CustomerID, in.Items)
	if err != nil {
		return nil, err
	}

	createdID, err := uc.idgen.New()
	if err != nil {
		return nil, err
	}
	authID, err := uc.idgen.New()
	if err != nil {
		return nil, err
	}

	domainEvents := []domain.Event{
		domain.NewEvent(createdID, domain.EvtOrderCreated, events.OrderCreatedPayload{
			OrderID:    order.ID,
			CustomerID: order.CustomerID,
			Items:      itemsToLines(order.Items),
			Total:      order.Total,
		}),
		domain.NewEvent(authID, domain.CmdAuthorizePayment, events.AuthorizePaymentPayload{
			OrderID:    order.ID,
			CustomerID: order.CustomerID,
			Amount:     order.Total,
		}),
	}

	if err := uc.repo.Create(ctx, order, domainEvents); err != nil {
		return nil, err
	}
	uc.log.Info().Str("order_id", order.ID).Int64("total", order.Total).Msg("order created")
	return order, nil
}

func itemsToLines(its []domain.Item) []events.OrderLine {
	out := make([]events.OrderLine, len(its))
	for i, it := range its {
		out[i] = events.OrderLine{SKU: it.SKU, Quantity: it.Quantity, Price: it.Price}
	}
	return out
}
