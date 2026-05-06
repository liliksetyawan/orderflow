// Order service composition root.
//
// This is where adapters are bound to ports — the only file in the service
// that knows about both. Boot sequence: load config → init observability →
// open downstream connections → run migrations → wire ports → start
// background workers (outbox dispatcher, saga consumer) → start HTTP server.
// Shutdown reverses the order, draining in-flight HTTP requests first.
package main

import (
	"context"
	"errors"
	"fmt"
	nethttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/liliksetyawan/orderflow/pkg/dbmigrate"
	"github.com/liliksetyawan/orderflow/pkg/idempotency"
	"github.com/liliksetyawan/orderflow/pkg/observability"
	"github.com/liliksetyawan/orderflow/pkg/outbox"
	pgxhelper "github.com/liliksetyawan/orderflow/pkg/pgx"
	"github.com/liliksetyawan/orderflow/pkg/rabbitmq"
	"github.com/liliksetyawan/orderflow/services/order/internal/adapter/consumer"
	httpadapter "github.com/liliksetyawan/orderflow/services/order/internal/adapter/http"
	"github.com/liliksetyawan/orderflow/services/order/internal/adapter/idgen"
	pgrepo "github.com/liliksetyawan/orderflow/services/order/internal/adapter/postgres"
	"github.com/liliksetyawan/orderflow/services/order/internal/app/usecase"
	"github.com/liliksetyawan/orderflow/services/order/internal/config"
	ordermigrations "github.com/liliksetyawan/orderflow/services/order/migrations"
)

const serviceName = "order-service"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger, shutdownTracer, err := observability.Init(rootCtx, serviceName, cfg.OTLPEndpoint)
	if err != nil {
		return fmt.Errorf("init observability: %w", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdownTracer(ctx)
	}()

	logger.Info().Msg("booting")

	pool, err := pgxhelper.Connect(rootCtx, pgxhelper.Config{
		Host:       cfg.PostgresHost,
		Port:       cfg.PostgresPort,
		User:       cfg.PostgresUser,
		Password:   cfg.PostgresPassword,
		Database:   cfg.PostgresDB,
		SearchPath: "order_svc",
	})
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	defer pool.Close()

	if err := dbmigrate.Run(rootCtx, pool, ordermigrations.FS); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	logger.Info().Msg("migrations up to date")

	mq, err := rabbitmq.Connect(rootCtx, cfg.RabbitMQURL)
	if err != nil {
		return fmt.Errorf("rabbitmq: %w", err)
	}
	defer observability.CloseOrLog(logger, "rabbitmq", mq)

	publisher, err := rabbitmq.NewPublisher(mq)
	if err != nil {
		return fmt.Errorf("rabbitmq publisher: %w", err)
	}
	defer observability.CloseOrLog(logger, "rabbitmq-publisher", publisher)

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer observability.CloseOrLog(logger, "redis", rdb)
	if err := rdb.Ping(rootCtx).Err(); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	idem := idempotency.New(rdb, 24*time.Hour)

	// Driven adapters → ports
	repo := pgrepo.New(pool)
	idGen := idgen.UUIDv7{}

	// Use cases (depend on ports only)
	createUC := usecase.NewCreateOrder(repo, idGen, logger)
	getUC := usecase.NewGetOrder(repo)
	sagaUC := usecase.NewSaga(repo, idGen, logger)

	// Background workers
	dispatcher := outbox.NewDispatcher(pool, publisher, logger)
	go dispatcher.Start(rootCtx)

	sagaCons := consumer.NewSaga(mq, sagaUC, idem, logger)
	if err := sagaCons.Start(rootCtx); err != nil {
		return fmt.Errorf("saga consumer: %w", err)
	}

	// Driving adapter: HTTP
	handler := httpadapter.NewOrderHandler(createUC, getUC, logger)
	srv := &nethttp.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           httpadapter.NewRouter(handler),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info().Int("port", cfg.HTTPPort).Msg("http server listening")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case <-rootCtx.Done():
		logger.Info().Msg("shutdown signal received")
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("http shutdown failed")
	}
	logger.Info().Msg("bye")
	return nil
}
