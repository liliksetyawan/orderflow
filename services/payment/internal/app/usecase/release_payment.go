package usecase

import (
	"context"
	"errors"

	"github.com/rs/zerolog"

	"github.com/lilik-setyawan/orderflow/pkg/events"
	"github.com/lilik-setyawan/orderflow/services/payment/internal/app/port"
	"github.com/lilik-setyawan/orderflow/services/payment/internal/domain"
)

type ReleasePayment struct {
	repo    port.PaymentRepository
	idgen   port.IDGenerator
	gateway port.PaymentGateway
	log     zerolog.Logger
}

func NewReleasePayment(repo port.PaymentRepository, idgen port.IDGenerator, gateway port.PaymentGateway, log zerolog.Logger) *ReleasePayment {
	return &ReleasePayment{
		repo:    repo,
		idgen:   idgen,
		gateway: gateway,
		log:     log.With().Str("usecase", "release_payment").Logger(),
	}
}

type ReleasePaymentInput struct {
	OrderID string
	Amount  int64 // informational; refund uses providerRef from the stored payment
}

// Execute is the handler for payment.release.v1 compensation commands.
//
// Idempotency: the use case is safe under at-least-once delivery. If the
// payment is already RELEASED, return success. If no payment exists for
// the order at all (compensation arrived before authorization landed in
// the DB — rare, but possible), return success — there's nothing to
// release. If status is FAILED or PENDING, log and return — releasing
// an unauthorized charge is meaningless.
func (uc *ReleasePayment) Execute(ctx context.Context, in ReleasePaymentInput) error {
	payment, err := uc.repo.GetByOrderID(ctx, in.OrderID)
	if errors.Is(err, domain.ErrNotFound) {
		uc.log.Info().Str("order_id", in.OrderID).Msg("no payment to release")
		return nil
	}
	if err != nil {
		return err
	}

	switch payment.Status {
	case domain.StatusReleased:
		return nil
	case domain.StatusFailed, domain.StatusPending:
		uc.log.Warn().Str("order_id", in.OrderID).Str("status", string(payment.Status)).
			Msg("cannot release in current status")
		return nil
	}

	if err := uc.gateway.Refund(ctx, payment.ProviderRef, payment.Amount); err != nil {
		uc.log.Warn().Err(err).Str("order_id", in.OrderID).Msg("refund transient error, requeuing")
		return err
	}

	if err := payment.Release(); err != nil {
		return err
	}

	evID, err := uc.idgen.New()
	if err != nil {
		return err
	}
	released := domain.NewEvent(evID, domain.EvtPaymentReleased, events.PaymentReleasedPayload{
		OrderID: payment.OrderID,
	})
	return uc.repo.Save(ctx, payment, []domain.Event{released})
}
