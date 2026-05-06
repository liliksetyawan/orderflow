package config

import "github.com/kelseyhightower/envconfig"

type Config struct {
	HTTPPort int `envconfig:"NOTIFICATION_HTTP_PORT" default:"8084"`

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
