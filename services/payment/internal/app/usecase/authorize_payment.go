// Package usecase contains the application services: orchestrations that
// fulfill product use cases by composing the domain over the ports.
package usecase

import (
	"context"
	"errors"

	"github.com/rs/zerolog"

	"github.com/liliksetyawan/orderflow/pkg/events"
	"github.com/liliksetyawan/orderflow/services/payment/internal/app/port"
	"github.com/liliksetyawan/orderflow/services/payment/internal/domain"
)

type AuthorizePayment struct {
	repo    port.PaymentRepository
	idgen   port.IDGenerator
	gateway port.PaymentGateway
	log     zerolog.Logger
}

func NewAuthorizePayment(repo port.PaymentRepository, idgen port.IDGenerator, gateway port.PaymentGateway, log zerolog.Logger) *AuthorizePayment {
	return &AuthorizePayment{
		repo:    repo,
		idgen:   idgen,
		gateway: gateway,
		log:     log.With().Str("usecase", "authorize_payment").Logger(),
	}
}

type AuthorizePaymentInput struct {
	OrderID    string
	CustomerID string
	Amount     int64
}

// Execute is the handler for payment.authorize.v1 commands. The flow is:
//
//  1. Idempotency fast-path: if a payment already exists for this order,
//     no-op return. (DB unique constraint is the slow-path guarantee.)
//  2. Call the gateway. Hard decline → mark FAILED + emit PaymentFailed.
//     Transient error → return error so the consumer requeues.
//  3. Persist the payment + emit event in one transaction (outbox).
func (uc *AuthorizePayment) Execute(ctx context.Context, in AuthorizePaymentInput) error {
	existing, err := uc.repo.GetByOrderID(ctx, in.OrderID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	if existing != nil {
		uc.log.Info().Str("order_id", in.OrderID).Str("status", string(existing.Status)).
			Msg("payment exists for order, idempotent skip")
		return nil
	}

	paymentID, err := uc.idgen.New()
	if err != nil {
		return err
	}
	payment, err := domain.NewPayment(paymentID, in.OrderID, in.CustomerID, in.Amount)
	if err != nil {
		return err
	}

	providerRef, gwErr := uc.gateway.Charge(ctx, in.OrderID, in.CustomerID, in.Amount)

	var domainEvents []domain.Event
	switch {
	case gwErr == nil:
		if err := payment.Authorize(providerRef); err != nil {
			return err
		}
		evID, err := uc.idgen.New()
		if err != nil {
			return err
		}
		domainEvents = append(domainEvents, domain.NewEvent(evID, domain.EvtPaymentAuthorized,
			events.PaymentAuthorizedPayload{
				OrderID:   payment.OrderID,
				PaymentID: payment.ID,
			}))

	case errors.Is(gwErr, port.ErrDeclined):
		if err := payment.Fail(); err != nil {
			return err
		}
		evID, err := uc.idgen.New()
		if err != nil {
			return err
		}
		domainEvents = append(domainEvents, domain.NewEvent(evID, domain.EvtPaymentFailed,
			events.PaymentFailedPayload{
				OrderID: payment.OrderID,
				Reason:  gwErr.Error(),
			}))

	default:
		// Transient: don't persist, let the consumer requeue and retry.
		uc.log.Warn().Err(gwErr).Str("order_id", in.OrderID).
			Msg("gateway transient error, returning to consumer")
		return gwErr
	}

	if err := uc.repo.Create(ctx, payment, domainEvents); err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			// Race against a parallel consumer instance: the other one
			// already inserted. Both did the work; only one persisted.
			// Acceptable — return success.
			uc.log.Info().Str("order_id", in.OrderID).Msg("payment created concurrently, ok")
			return nil
		}
		return err
	}

	uc.log.Info().
		Str("order_id", payment.OrderID).
		Str("payment_id", payment.ID).
		Str("status", string(payment.Status)).
		Msg("payment processed")
	return nil
}
