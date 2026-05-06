// Package postgres is the driven adapter implementing
// port.NotificationRepository. UNIQUE(order_id, type, channel) collisions
// are surfaced as domain.ErrAlreadyExists so the use case can treat them
// as a successful idempotent skip.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lilik-setyawan/orderflow/services/notification/internal/app/port"
	"github.com/lilik-setyawan/orderflow/services/notification/internal/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

var _ port.NotificationRepository = (*Repository)(nil)

func (r *Repository) Create(ctx context.Context, n *domain.Notification) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO notifications (id, order_id, customer_id, type, channel, sent_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, n.ID, n.OrderID, n.CustomerID, n.Type, n.Channel, n.SentAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("insert notification: %w", err)
	}
	return nil
}

func (r *Repository) GetByOrderTypeChannel(ctx context.Context, orderID, typ, channel string) (*domain.Notification, error) {
	var n domain.Notification
	err := r.pool.QueryRow(ctx, `
		SELECT id, order_id, customer_id, type, channel, sent_at
		  FROM notifications
		 WHERE order_id = $1 AND type = $2 AND channel = $3
	`, orderID, typ, channel).Scan(&n.ID, &n.OrderID, &n.CustomerID, &n.Type, &n.Channel, &n.SentAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select notification: %w", err)
	}
	return &n, nil
}
