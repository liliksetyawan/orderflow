// Package usecase contains the application services for the inventory
// service: orchestrations that compose the domain over the ports.
package usecase

import (
	"context"
	"errors"

	"github.com/rs/zerolog"

	"github.com/liliksetyawan/orderflow/pkg/events"
	"github.com/liliksetyawan/orderflow/services/inventory/internal/app/port"
	"github.com/liliksetyawan/orderflow/services/inventory/internal/domain"
)

type ReserveInventory struct {
	repo  port.ReservationRepository
	idgen port.IDGenerator
	log   zerolog.Logger
}

func NewReserveInventory(repo port.ReservationRepository, idgen port.IDGenerator, log zerolog.Logger) *ReserveInventory {
	return &ReserveInventory{
		repo:  repo,
		idgen: idgen,
		log:   log.With().Str("usecase", "reserve_inventory").Logger(),
	}
}

type ReserveInventoryInput struct {
	OrderID string
	Items   []domain.Item
}

// Execute is the handler for inventory.reserve.v1. Flow:
//
//  1. Idempotency fast-path: if a reservation already exists (RESERVED or
//     FAILED), no-op return.
//  2. Try ReserveAtomic. Success → done; outbox publishes inventory.reserved.
//  3. ErrInsufficientStock → record FAILED reservation + outbox failed event.
//  4. ErrAlreadyExists (race) → no-op.
//  5. Anything else → return error so consumer requeues.
func (uc *ReserveInventory) Execute(ctx context.Context, in ReserveInventoryInput) error {
	existing, err := uc.repo.GetByOrderID(ctx, in.OrderID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	if existing != nil {
		uc.log.Info().Str("order_id", in.OrderID).Str("status", string(existing.Status)).
			Msg("reservation exists, idempotent skip")
		return nil
	}

	resID, err := uc.idgen.New()
	if err != nil {
		return err
	}
	res, err := domain.New(resID, in.OrderID, in.Items)
	if err != nil {
		return err
	}
	if err := res.MarkReserved(); err != nil {
		return err
	}

	evID, err := uc.idgen.New()
	if err != nil {
		return err
	}
	reservedEv := domain.NewEvent(evID, domain.EvtInventoryReserved, events.InventoryReservedPayload{
		OrderID:       res.OrderID,
		ReservationID: res.ID,
	})

	err = uc.repo.ReserveAtomic(ctx, res, []domain.Event{reservedEv})
	if err == nil {
		uc.log.Info().Str("order_id", res.OrderID).Str("reservation_id", res.ID).
			Msg("inventory reserved")
		return nil
	}
	if errors.Is(err, domain.ErrAlreadyExists) {
		uc.log.Info().Str("order_id", in.OrderID).Msg("reservation created concurrently, ok")
		return nil
	}
	if !errors.Is(err, domain.ErrInsufficientStock) {
		// Transient: bubble up so consumer requeues
		return err
	}

	// Insufficient stock: record FAILED reservation + emit failure event
	failedID, err := uc.idgen.New()
	if err != nil {
		return err
	}
	failed, err := domain.New(failedID, in.OrderID, in.Items)
	if err != nil {
		return err
	}
	const reason = "insufficient stock"
	if err := failed.MarkFailed(reason); err != nil {
		return err
	}

	failedEvID, err := uc.idgen.New()
	if err != nil {
		return err
	}
	failedEv := domain.NewEvent(failedEvID, domain.EvtInventoryFailed, events.InventoryFailedPayload{
		OrderID: in.OrderID,
		Reason:  reason,
	})

	if err := uc.repo.RecordFailed(ctx, failed, []domain.Event{failedEv}); err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			return nil
		}
		return err
	}
	uc.log.Info().Str("order_id", in.OrderID).Msg("reservation failed: insufficient stock")
	return nil
}
