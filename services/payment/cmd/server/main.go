// Payment service composition root.
//
// No client-facing HTTP API: the service is driven entirely by AMQP commands
// (payment.authorize.v1, payment.release.v1). HTTP exposes /health only.
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
	"github.com/liliksetyawan/orderflow/services/payment/internal/adapter/consumer"
	mockgw "github.com/liliksetyawan/orderflow/services/payment/internal/adapter/gateway"
	httpadapter "github.com/liliksetyawan/orderflow/services/payment/internal/adapter/http"
	"github.com/liliksetyawan/orderflow/services/payment/internal/adapter/idgen"
	pgrepo "github.com/liliksetyawan/orderflow/services/payment/internal/adapter/postgres"
	"github.com/liliksetyawan/orderflow/services/payment/internal/app/usecase"
	"github.com/liliksetyawan/orderflow/services/payment/internal/config"
	paymentmigrations "github.com/liliksetyawan/orderflow/services/payment/migrations"
)

const serviceName = "payment-service"

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
		SearchPath: "payment_svc",
	})
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	defer pool.Close()

	if err := dbmigrate.Run(rootCtx, pool, paymentmigrations.FS); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	logger.Info().Msg("migrations up to date")

	mq, err := rabbitmq.Connect(rootCtx, cfg.RabbitMQURL)
	if err != nil {
		return fmt.Errorf("rabbitmq: %w", err)
	}
	defer mq.Close()

	publisher, err := rabbitmq.NewPublisher(mq)
	if err != nil {
		return fmt.Errorf("rabbitmq publisher: %w", err)
	}
	defer publisher.Close()

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer rdb.Close()
	if err := rdb.Ping(rootCtx).Err(); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	idem := idempotency.New(rdb, 24*time.Hour)

	// Driven adapters → ports
	repo := pgrepo.New(pool)
	idGen := idgen.UUIDv7{}
	gateway := mockgw.NewMock(cfg.GatewaySuccessRate)

	// Use cases
	authorizeUC := usecase.NewAuthorizePayment(repo, idGen, gateway, logger)
	releaseUC := usecase.NewReleasePayment(repo, idGen, gateway, logger)

	// Background workers
	dispatcher := outbox.NewDispatcher(pool, publisher, logger)
	go dispatcher.Start(rootCtx)

	cmdCons := consumer.New(mq, authorizeUC, releaseUC, idem, logger)
	if err := cmdCons.Start(rootCtx); err != nil {
		return fmt.Errorf("payment consumer: %w", err)
	}

	// Driving adapter: HTTP (operational only)
	srv := &nethttp.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           httpadapter.NewRouter(),
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
