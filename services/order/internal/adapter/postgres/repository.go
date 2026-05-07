// Package postgres is the driven adapter implementing port.OrderRepository
// against PostgreSQL via pgx. It writes the order, its items, and any
// accompanying outbox rows in a single transaction — that's how we get
// "save state and publish events atomically" without a distributed
// transaction.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/liliksetyawan/orderflow/services/order/internal/app/port"
	"github.com/liliksetyawan/orderflow/services/order/internal/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// Compile-time check.
var _ port.OrderRepository = (*Repository)(nil)

func (r *Repository) Create(ctx context.Context, o *domain.Order, evs []domain.Event) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO orders (id, customer_id, total, status, created_at, updated_at, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, o.ID, o.CustomerID, o.Total, string(o.Status), o.CreatedAt, o.UpdatedAt, o.Version); err != nil {
		return fmt.Errorf("insert order: %w", err)
	}
	for _, it := range o.Items {
		if _, err := tx.Exec(ctx, `
			INSERT INTO order_items (order_id, sku, quantity, price)
			VALUES ($1, $2, $3, $4)
		`, o.ID, it.SKU, it.Quantity, it.Price); err != nil {
			return fmt.Errorf("insert order_item: %w", err)
		}
	}
	if err := writeOutbox(ctx, tx, evs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Save updates an existing order with optimistic concurrency. If another
// writer bumped the version since we read it, this returns ErrConcurrentUpdate
// — caller should reload and retry.
func (r *Repository) Save(ctx context.Context, o *domain.Order, evs []domain.Event) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	cmd, err := tx.Exec(ctx, `
		UPDATE orders
		   SET status     = $1,
		       updated_at = $2,
		       version    = version + 1
		 WHERE id      = $3
		   AND version = $4
	`, string(o.Status), o.UpdatedAt, o.ID, o.Version)
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrConcurrentUpdate
	}

	if err := writeOutbox(ctx, tx, evs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) List(ctx context.Context, status string, limit, offset int) ([]*domain.Order, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// $1 doubles as "no filter" sentinel: empty string matches every row.
	const countQ = `SELECT COUNT(*) FROM orders WHERE ($1 = '' OR status = $1)`
	const listQ = `
		SELECT id, customer_id, total, status, created_at, updated_at, version
		  FROM orders
		 WHERE ($1 = '' OR status = $1)
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3
	`

	var total int
	if err := r.pool.QueryRow(ctx, countQ, status).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}

	rows, err := r.pool.Query(ctx, listQ, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		var (
			o         domain.Order
			statusStr string
		)
		if err := rows.Scan(&o.ID, &o.CustomerID, &o.Total, &statusStr,
			&o.CreatedAt, &o.UpdatedAt, &o.Version); err != nil {
			return nil, 0, fmt.Errorf("scan order: %w", err)
		}
		o.Status = domain.OrderStatus(statusStr)
		orders = append(orders, &o)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate orders: %w", err)
	}
	return orders, total, nil
}

func (r *Repository) Get(ctx context.Context, id string) (*domain.Order, error) {
	var (
		o         domain.Order
		statusStr string
	)
	err := r.pool.QueryRow(ctx, `
		SELECT id, customer_id, total, status, created_at, updated_at, version
		  FROM orders
		 WHERE id = $1
	`, id).Scan(&o.ID, &o.CustomerID, &o.Total, &statusStr, &o.CreatedAt, &o.UpdatedAt, &o.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select order: %w", err)
	}
	o.Status = domain.OrderStatus(statusStr)

	rows, err := r.pool.Query(ctx, `
		SELECT sku, quantity, price FROM order_items WHERE order_id = $1
	`, id)
	if err != nil {
		return nil, fmt.Errorf("select items: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var it domain.Item
		if err := rows.Scan(&it.SKU, &it.Quantity, &it.Price); err != nil {
			return nil, err
		}
		o.Items = append(o.Items, it)
	}
	return &o, nil
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
