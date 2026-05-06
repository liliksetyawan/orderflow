package usecase

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/liliksetyawan/orderflow/pkg/events"
	"github.com/liliksetyawan/orderflow/services/order/internal/app/port"
	"github.com/liliksetyawan/orderflow/services/order/internal/domain"
)

// Saga is the orchestrator: each method is a use case triggered by a reply
// event from a downstream service. Each one loads the order, asks the domain
// to transition state, and emits the next-step events transactionally.
//
// State guards: every handler short-circuits if the order has already moved
// past the source state. This makes handlers safe under at-least-once
// delivery — duplicate redelivery becomes a no-op rather than a transition
// error that would loop forever.
type Saga struct {
	repo  port.OrderRepository
	idgen port.IDGenerator
	log   zerolog.Logger
}

func NewSaga(repo port.OrderRepository, idgen port.IDGenerator, log zerolog.Logger) *Saga {
	return &Saga{
		repo:  repo,
		idgen: idgen,
		log:   log.With().Str("usecase", "saga").Logger(),
	}
}

// OnPaymentAuthorized: PENDING → AUTHORIZED, then command inventory to reserve.
func (s *Saga) OnPaymentAuthorized(ctx context.Context, p events.PaymentAuthorizedPayload) error {
	order, err := s.repo.Get(ctx, p.OrderID)
	if err != nil {
		return err
	}
	if order.Status != domain.StatusPending {
		s.log.Info().Str("order_id", order.ID).Str("status", string(order.Status)).
			Msg("payment.authorized for non-PENDING order, skipping")
		return nil
	}
	if err := order.MarkAuthorized(); err != nil {
		return err
	}

	cmdID, err := s.idgen.New()
	if err != nil {
		return err
	}
	cmd := domain.NewEvent(cmdID, domain.CmdReserveInventory, events.ReserveInventoryPayload{
		OrderID: order.ID,
		Items:   itemsToLines(order.Items),
	})
	return s.repo.Save(ctx, order, []domain.Event{cmd})
}

// OnPaymentFailed: PENDING → CANCELED, emit OrderCanceled (notification).
func (s *Saga) OnPaymentFailed(ctx context.Context, p events.PaymentFailedPayload) error {
	order, err := s.repo.Get(ctx, p.OrderID)
	if err != nil {
		return err
	}
	if order.Status != domain.StatusPending {
		s.log.Info().Str("order_id", order.ID).Str("status", string(order.Status)).
			Msg("payment.failed for non-PENDING order, skipping")
		return nil
	}
	if err := order.Cancel(); err != nil {
		return err
	}

	evID, err := s.idgen.New()
	if err != nil {
		return err
	}
	canceled := domain.NewEvent(evID, domain.EvtOrderCanceled, events.OrderCanceledPayload{
		OrderID:    order.ID,
		CustomerID: order.CustomerID,
		Reason:     "payment_failed: " + p.Reason,
	})
	return s.repo.Save(ctx, order, []domain.Event{canceled})
}

// OnInventoryReserved: AUTHORIZED → CONFIRMED, emit OrderConfirmed.
func (s *Saga) OnInventoryReserved(ctx context.Context, p events.InventoryReservedPayload) error {
	order, err := s.repo.Get(ctx, p.OrderID)
	if err != nil {
		return err
	}
	if order.Status != domain.StatusAuthorized {
		s.log.Info().Str("order_id", order.ID).Str("status", string(order.Status)).
			Msg("inventory.reserved for non-AUTHORIZED order, skipping")
		return nil
	}
	if err := order.MarkConfirmed(); err != nil {
		return err
	}

	evID, err := s.idgen.New()
	if err != nil {
		return err
	}
	confirmed := domain.NewEvent(evID, domain.EvtOrderConfirmed, events.OrderConfirmedPayload{
		OrderID:    order.ID,
		CustomerID: order.CustomerID,
	})
	return s.repo.Save(ctx, order, []domain.Event{confirmed})
}

// OnInventoryFailed: AUTHORIZED → CANCELED, plus compensating ReleasePayment
// command, plus OrderCanceled event. All three persist + emit atomically.
func (s *Saga) OnInventoryFailed(ctx context.Context, p events.InventoryFailedPayload) error {
	order, err := s.repo.Get(ctx, p.OrderID)
	if err != nil {
		return err
	}
	if order.Status != domain.StatusAuthorized {
		s.log.Info().Str("order_id", order.ID).Str("status", string(order.Status)).
			Msg("inventory.failed for non-AUTHORIZED order, skipping")
		return nil
	}
	if err := order.Cancel(); err != nil {
		return err
	}

	relID, err := s.idgen.New()
	if err != nil {
		return err
	}
	canID, err := s.idgen.New()
	if err != nil {
		return err
	}

	releaseCmd := domain.NewEvent(relID, domain.CmdReleasePayment, events.ReleasePaymentPayload{
		OrderID: order.ID,
		Amount:  order.Total,
	})
	canceled := domain.NewEvent(canID, domain.EvtOrderCanceled, events.OrderCanceledPayload{
		OrderID:    order.ID,
		CustomerID: order.CustomerID,
		Reason:     "inventory_failed: " + p.Reason,
	})
	return s.repo.Save(ctx, order, []domain.Event{releaseCmd, canceled})
}
