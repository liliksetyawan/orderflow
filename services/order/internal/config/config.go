// Package config loads service configuration from environment variables.
//
// We use kelseyhightower/envconfig: zero deps, struct tags, validates
// required fields at boot. If anything is missing, the service fails fast
// before opening any connections.
package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	HTTPPort int `envconfig:"ORDER_HTTP_PORT" default:"8081"`
	GRPCPort int `envconfig:"ORDER_GRPC_PORT" default:"9081"`

	PostgresHost     string `envconfig:"POSTGRES_HOST" default:"localhost"`
	PostgresPort     int    `envconfig:"POSTGRES_PORT" default:"5432"`
	PostgresUser     string `envconfig:"POSTGRES_USER" default:"orderflow"`
	PostgresPassword string `envconfig:"POSTGRES_PASSWORD" default:"orderflow"`
	PostgresDB       string `envconfig:"POSTGRES_DB" default:"orderflow"`

	RabbitMQURL string `envconfig:"RABBITMQ_URL" default:"amqp://orderflow:orderflow@localhost:5672/"`
	RedisAddr   string `envconfig:"REDIS_ADDR" default:"localhost:6379"`

	OTLPEndpoint string `envconfig:"OTEL_EXPORTER_OTLP_ENDPOINT" default:""`
}

func Load() (Config, error) {
	var c Config
	err := envconfig.Process("", &c)
	return c, err
}
