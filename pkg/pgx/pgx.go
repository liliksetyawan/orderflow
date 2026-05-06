// Package pgx wraps pgxpool with sane defaults so every service connects the
// same way. Each service overrides the search_path so its tables live in its
// own schema (order_svc, payment_svc, ...).
package pgx

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Host       string
	Port       int
	User       string
	Password   string
	Database   string
	SearchPath string // schema name, e.g. "order_svc"
}

func (c Config) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable&search_path=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.SearchPath,
	)
}

// Connect returns a pool with conservative defaults; caller closes via Close().
func Connect(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	pcfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse pg config: %w", err)
	}
	pcfg.MaxConns = 20
	pcfg.MinConns = 2
	pcfg.MaxConnLifetime = 30 * time.Minute
	pcfg.MaxConnIdleTime = 5 * time.Minute
	pcfg.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, pcfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}
