// Package usecase contains the application services for the notification
// service.
package usecase

import (
	"context"
	"errors"

	"github.com/rs/zerolog"

	"github.com/liliksetyawan/orderflow/services/notification/internal/app/port"
	"github.com/liliksetyawan/orderflow/services/notification/internal/domain"
)

type SendNotification struct {
	repo     port.NotificationRepository
	notifier port.Notifier
	idgen    port.IDGenerator
	log      zerolog.Logger
}

func NewSendNotification(repo port.NotificationRepository, notifier port.Notifier, idgen port.IDGenerator, log zerolog.Logger) *SendNotification {
	return &SendNotification{
		repo:     repo,
		notifier: notifier,
		idgen:    idgen,
		log:      log.With().Str("usecase", "send_notification").Logger(),
	}
}

type SendNotificationInput struct {
	OrderID    string
	CustomerID string
	Type       string // domain.TypeOrderConfirmed or domain.TypeOrderCanceled
	Reason     string // populated for OrderCanceled
}

// Execute is the handler invoked by the consumer for each terminal order
// event. Flow:
//
//  1. Idempotency fast-path: if a notification record already exists for
//     this (order, type, channel), no-op.
//  2. Send via the injected Notifier. Send failure → return error so the
//     consumer requeues; nothing is persisted, retry will resend.
//  3. Persist the success record. UNIQUE constraint catches racing
//     duplicates from parallel consumer instances.
//
// Trade-off: if the process crashes between Send and Create, a retry
// will resend (rare duplicate). For log-channel demos this is fine; real
// channels should use provider-side idempotency keys.
func (uc *SendNotification) Execute(ctx context.Context, in SendNotificationInput) error {
	channel := uc.notifier.Channel()

	existing, err := uc.repo.GetByOrderTypeChannel(ctx, in.OrderID, in.Type, channel)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	if existing != nil {
		uc.log.Debug().Str("order_id", in.OrderID).Str("type", in.Type).
			Msg("notification already sent, idempotent skip")
		return nil
	}

	id, err := uc.idgen.New()
	if err != nil {
		return err
	}
	n, err := domain.New(id, in.OrderID, in.CustomerID, in.Type, channel)
	if err != nil {
		return err
	}

	if err := uc.notifier.Send(ctx, *n, in.Reason); err != nil {
		uc.log.Warn().Err(err).Str("order_id", in.OrderID).Msg("send failed, will requeue")
		return err
	}

	if err := uc.repo.Create(ctx, n); err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			return nil // race won by parallel instance
		}
		uc.log.Warn().Err(err).Str("order_id", in.OrderID).
			Msg("send succeeded but record failed; retry may produce duplicate")
		return err
	}
	uc.log.Info().Str("order_id", n.OrderID).Str("type", n.Type).Str("channel", channel).
		Msg("notification recorded")
	return nil
}
