// Package postgres is the driven adapter implementing port.PaymentRepository.
// It writes the payment row + outbox rows in a single transaction; that
// atomicity is what gives us the "save state and publish events together"
// guarantee without a distributed transaction.
//
// Idempotency: payments.order_id has a UNIQUE constraint. A duplicate INSERT
// surfaces as Postgres SQLSTATE 23505, which we translate to
// domain.ErrAlreadyExists for the use case.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lilik-setyawan/orderflow/services/payment/internal/app/port"
	"github.com/lilik-setyawan/orderflow/services/payment/internal/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

var _ port.PaymentRepository = (*Repository)(nil)

func (r *Repository) Create(ctx context.Context, p *domain.Payment, evs []domain.Event) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var providerRef *string
	if p.ProviderRef != "" {
		ref := p.ProviderRef
		providerRef = &ref
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO payments (id, order_id, customer_id, amount, status, provider_ref, created_at, updated_at, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, p.ID, p.OrderID, p.CustomerID, p.Amount, string(p.Status), providerRef,
		p.CreatedAt, p.UpdatedAt, p.Version); err != nil {

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("insert payment: %w", err)
	}

	if err := writeOutbox(ctx, tx, evs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) Save(ctx context.Context, p *domain.Payment, evs []domain.Event) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var providerRef *string
	if p.ProviderRef != "" {
		ref := p.ProviderRef
		providerRef = &ref
	}

	cmd, err := tx.Exec(ctx, `
		UPDATE payments
		   SET status       = $1,
		       provider_ref = $2,
		       updated_at   = $3,
		       version      = version + 1
		 WHERE id      = $4
		   AND version = $5
	`, string(p.Status), providerRef, p.UpdatedAt, p.ID, p.Version)
	if err != nil {
		return fmt.Errorf("update payment: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrConcurrentUpdate
	}

	if err := writeOutbox(ctx, tx, evs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	var (
		p           domain.Payment
		statusStr   string
		providerRef *string
	)
	err := r.pool.QueryRow(ctx, `
		SELECT id, order_id, customer_id, amount, status, provider_ref, created_at, updated_at, version
		  FROM payments
		 WHERE order_id = $1
	`, orderID).Scan(&p.ID, &p.OrderID, &p.CustomerID, &p.Amount, &statusStr,
		&providerRef, &p.CreatedAt, &p.UpdatedAt, &p.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select payment: %w", err)
	}
	p.Status = domain.PaymentStatus(statusStr)
	if providerRef != nil {
		p.ProviderRef = *providerRef
	}
	return &p, nil
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
