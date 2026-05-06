// Package postgres is the driven adapter implementing
// port.ReservationRepository. The atomic methods do their entire job
// (reservation + items + stock + outbox) in one transaction, which is
// the heart of the saga's correctness guarantee.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lilik-setyawan/orderflow/services/inventory/internal/app/port"
	"github.com/lilik-setyawan/orderflow/services/inventory/internal/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

var _ port.ReservationRepository = (*Repository)(nil)

func (r *Repository) ReserveAtomic(ctx context.Context, res *domain.Reservation, evs []domain.Event) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := insertReservation(ctx, tx, res); err != nil {
		return err
	}
	if err := insertItems(ctx, tx, res); err != nil {
		return err
	}

	// Conditional decrement: if any SKU lacks stock, returns
	// ErrInsufficientStock and the deferred Rollback unwinds everything
	// (reservation insert, items insert, prior decrements) atomically.
	for _, item := range res.Items {
		cmd, err := tx.Exec(ctx, `
			UPDATE stocks
			   SET quantity = quantity - $1,
			       version  = version + 1
			 WHERE sku      = $2
			   AND quantity >= $1
		`, item.Quantity, item.SKU)
		if err != nil {
			return fmt.Errorf("update stock: %w", err)
		}
		if cmd.RowsAffected() == 0 {
			return domain.ErrInsufficientStock
		}
	}

	if err := writeOutbox(ctx, tx, evs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) RecordFailed(ctx context.Context, res *domain.Reservation, evs []domain.Event) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := insertReservation(ctx, tx, res); err != nil {
		return err
	}
	if err := insertItems(ctx, tx, res); err != nil {
		return err
	}
	if err := writeOutbox(ctx, tx, evs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) ReleaseAtomic(ctx context.Context, res *domain.Reservation, evs []domain.Event) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Increment stocks back. We don't gate on version here — releases are
	// always safe to apply (we already decremented when reserving).
	for _, item := range res.Items {
		if _, err := tx.Exec(ctx, `
			UPDATE stocks
			   SET quantity = quantity + $1,
			       version  = version + 1
			 WHERE sku = $2
		`, item.Quantity, item.SKU); err != nil {
			return fmt.Errorf("restore stock: %w", err)
		}
	}

	// Optimistic update of the reservation row.
	cmd, err := tx.Exec(ctx, `
		UPDATE reservations
		   SET status     = $1,
		       updated_at = $2,
		       version    = version + 1
		 WHERE id      = $3
		   AND version = $4
	`, string(res.Status), res.UpdatedAt, res.ID, res.Version)
	if err != nil {
		return fmt.Errorf("update reservation: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrConcurrentUpdate
	}

	if err := writeOutbox(ctx, tx, evs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) GetByOrderID(ctx context.Context, orderID string) (*domain.Reservation, error) {
	var (
		res       domain.Reservation
		statusStr string
		reason    *string
	)
	err := r.pool.QueryRow(ctx, `
		SELECT id, order_id, status, reason, created_at, updated_at, version
		  FROM reservations
		 WHERE order_id = $1
	`, orderID).Scan(&res.ID, &res.OrderID, &statusStr, &reason,
		&res.CreatedAt, &res.UpdatedAt, &res.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select reservation: %w", err)
	}
	res.Status = domain.ReservationStatus(statusStr)
	if reason != nil {
		res.Reason = *reason
	}

	rows, err := r.pool.Query(ctx, `
		SELECT sku, quantity FROM reservation_items WHERE reservation_id = $1
	`, res.ID)
	if err != nil {
		return nil, fmt.Errorf("select items: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var it domain.Item
		if err := rows.Scan(&it.SKU, &it.Quantity); err != nil {
			return nil, err
		}
		res.Items = append(res.Items, it)
	}
	return &res, nil
}

// --- shared tx helpers ---

func insertReservation(ctx context.Context, tx pgx.Tx, res *domain.Reservation) error {
	var reason *string
	if res.Reason != "" {
		s := res.Reason
		reason = &s
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO reservations (id, order_id, status, reason, created_at, updated_at, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, res.ID, res.OrderID, string(res.Status), reason, res.CreatedAt, res.UpdatedAt, res.Version)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("insert reservation: %w", err)
	}
	return nil
}

func insertItems(ctx context.Context, tx pgx.Tx, res *domain.Reservation) error {
	for _, it := range res.Items {
		if _, err := tx.Exec(ctx, `
			INSERT INTO reservation_items (reservation_id, sku, quantity)
			VALUES ($1, $2, $3)
		`, res.ID, it.SKU, it.Quantity); err != nil {
			return fmt.Errorf("insert reservation_item: %w", err)
		}
	}
	return nil
}

func writeOutbox(ctx context.Context, tx pgx.Tx, evs []domain.Event) error {
	for _, ev := range evs {
		rec, err := toOutboxRecord(ev)
		if err != nil {
			return fmt.Errorf("outbox mapping: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO outbox (id, routing_key, payload) VALUES ($1, $2, $3)
		`, rec.ID, rec.RoutingKey, rec.Body); err != nil {
			return fmt.Errorf("insert outbox: %w", err)
		}
	}
	return nil
}
