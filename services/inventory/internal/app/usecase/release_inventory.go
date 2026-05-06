package usecase

import (
	"context"
	"errors"

	"github.com/rs/zerolog"

	"github.com/lilik-setyawan/orderflow/pkg/events"
	"github.com/lilik-setyawan/orderflow/services/inventory/internal/app/port"
	"github.com/lilik-setyawan/orderflow/services/inventory/internal/domain"
)

type ReleaseInventory struct {
	repo  port.ReservationRepository
	idgen port.IDGenerator
	log   zerolog.Logger
}

func NewReleaseInventory(repo port.ReservationRepository, idgen port.IDGenerator, log zerolog.Logger) *ReleaseInventory {
	return &ReleaseInventory{
		repo:  repo,
		idgen: idgen,
		log:   log.With().Str("usecase", "release_inventory").Logger(),
	}
}

type ReleaseInventoryInput struct {
	OrderID string
}

// Execute is the handler for inventory.release.v1 compensation commands.
// Idempotency: terminal states (RELEASED, FAILED) are no-ops; missing
// reservation is also a no-op (release arrived before reserve landed,
// or never reserved at all).
func (uc *ReleaseInventory) Execute(ctx context.Context, in ReleaseInventoryInput) error {
	res, err := uc.repo.GetByOrderID(ctx, in.OrderID)
	if errors.Is(err, domain.ErrNotFound) {
		uc.log.Info().Str("order_id", in.OrderID).Msg("no reservation to release")
		return nil
	}
	if err != nil {
		return err
	}

	switch res.Status {
	case domain.StatusReleased:
		return nil
	case domain.StatusFailed:
		uc.log.Warn().Str("order_id", in.OrderID).Msg("cannot release a FAILED reservation")
		return nil
	case domain.StatusReserved:
		// proceed
	default:
		uc.log.Warn().Str("order_id", in.OrderID).Str("status", string(res.Status)).
			Msg("unexpected reservation status, skipping")
		return nil
	}

	if err := res.MarkReleased(); err != nil {
		return err
	}

	evID, err := uc.idgen.New()
	if err != nil {
		return err
	}
	releasedEv := domain.NewEvent(evID, domain.EvtInventoryReleased, events.InventoryReleasedPayload{
		OrderID: res.OrderID,
	})
	return uc.repo.ReleaseAtomic(ctx, res, []domain.Event{releasedEv})
}
