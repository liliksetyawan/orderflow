// Package outbox implements the transactional outbox pattern.
//
// The producer writes business state and an outbox row in the same DB
// transaction, guaranteeing they commit or fail atomically. A separate
// dispatcher polls the outbox table, publishes pending rows to RabbitMQ
// (with publisher confirms), and marks them sent. Crash between commit
// and publish? The dispatcher picks up where it left off. No lost events,
// no event-without-state-change.
//
// Multiple dispatcher instances are safe to run in parallel — claims use
// SELECT ... FOR UPDATE SKIP LOCKED.
package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/lilik-setyawan/orderflow/pkg/events"
)

// Schema is the SQL for the outbox table. Embed verbatim in service migrations.
const Schema = `
CREATE TABLE IF NOT EXISTS outbox (
    id           UUID PRIMARY KEY,
    routing_key  TEXT        NOT NULL,
    payload      JSONB       NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at      TIMESTAMPTZ,
    attempts     INT         NOT NULL DEFAULT 0,
    last_error   TEXT
);
CREATE INDEX IF NOT EXISTS idx_outbox_unsent
    ON outbox (created_at) WHERE sent_at IS NULL;
`

// Publisher is the minimum the dispatcher needs. Implemented by
// pkg/rabbitmq.Publisher; mockable in tests.
type Publisher interface {
	Publish(ctx context.Context, routingKey, msgID string, body []byte) error
}

// Write inserts an outbox row inside an existing transaction. Caller commits.
func Write(ctx context.Context, tx pgx.Tx, routingKey string, env events.Envelope) error {
	payload, err := env.Marshal()
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO outbox (id, routing_key, payload) VALUES ($1, $2, $3)`,
		env.ID, routingKey, payload,
	)
	return err
}

type Dispatcher struct {
	pool      *pgxpool.Pool
	publisher Publisher
	logger    zerolog.Logger
	interval  time.Duration
	batch     int
}

func NewDispatcher(pool *pgxpool.Pool, pub Publisher, logger zerolog.Logger) *Dispatcher {
	return &Dispatcher{
		pool:      pool,
		publisher: pub,
		logger:    logger.With().Str("component", "outbox").Logger(),
		interval:  500 * time.Millisecond,
		batch:     50,
	}
}

func (d *Dispatcher) Start(ctx context.Context) {
	t := time.NewTicker(d.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			d.logger.Info().Msg("dispatcher stopping")
			return
		case <-t.C:
			if err := d.tick(ctx); err != nil {
				d.logger.Error().Err(err).Msg("dispatcher tick failed")
			}
		}
	}
}

func (d *Dispatcher) tick(ctx context.Context) error {
	tx, err := d.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		SELECT id, routing_key, payload
		  FROM outbox
		 WHERE sent_at IS NULL
		 ORDER BY created_at
		 LIMIT $1
		 FOR UPDATE SKIP LOCKED
	`, d.batch)
	if err != nil {
		return fmt.Errorf("select: %w", err)
	}

	type row struct {
		id, routingKey string
		payload        []byte
	}
	var batch []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.routingKey, &r.payload); err != nil {
			rows.Close()
			return fmt.Errorf("scan: %w", err)
		}
		batch = append(batch, r)
	}
	rows.Close()
	if len(batch) == 0 {
		return tx.Commit(ctx)
	}

	for _, r := range batch {
		// MessageId == event id → consumers use it as idempotency key.
		if err := d.publisher.Publish(ctx, r.routingKey, r.id, r.payload); err != nil {
			_, _ = tx.Exec(ctx,
				`UPDATE outbox SET attempts = attempts + 1, last_error = $2 WHERE id = $1`,
				r.id, err.Error())
			continue
		}
		if _, err := tx.Exec(ctx,
			`UPDATE outbox SET sent_at = now() WHERE id = $1`, r.id); err != nil {
			return fmt.Errorf("mark sent: %w", err)
		}
	}
	return tx.Commit(ctx)
}
