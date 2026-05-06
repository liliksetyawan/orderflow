// Package consumer is the driving adapter that subscribes to RabbitMQ for
// payment.* commands and dispatches to the matching use case. It uses the
// post-mark idempotency pattern from pkg/idempotency.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"

	"github.com/lilik-setyawan/orderflow/pkg/events"
	"github.com/lilik-setyawan/orderflow/pkg/idempotency"
	"github.com/lilik-setyawan/orderflow/pkg/rabbitmq"
	"github.com/lilik-setyawan/orderflow/services/payment/internal/app/usecase"
)

const (
	queueName    = "payment.commands"
	consumerTag  = "payment-svc"
	prefetchSize = 32
)

var routingKeys = []string{
	"payment.authorize.v1",
	"payment.release.v1",
}

type CommandConsumer struct {
	conn      *rabbitmq.Connection
	authorize *usecase.AuthorizePayment
	release   *usecase.ReleasePayment
	idem      *idempotency.Cache
	log       zerolog.Logger
}

func New(conn *rabbitmq.Connection, authorize *usecase.AuthorizePayment, release *usecase.ReleasePayment, idem *idempotency.Cache, log zerolog.Logger) *CommandConsumer {
	return &CommandConsumer{
		conn:      conn,
		authorize: authorize,
		release:   release,
		idem:      idem,
		log:       log.With().Str("component", "payment-consumer").Logger(),
	}
}

func (c *CommandConsumer) Start(ctx context.Context) error {
	if err := c.conn.DeclareQueue(queueName, routingKeys); err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}
	ch, err := c.conn.Raw().Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	if err := ch.Qos(prefetchSize, 0, false); err != nil {
		return fmt.Errorf("qos: %w", err)
	}
	msgs, err := ch.Consume(queueName, consumerTag, false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	go func() {
		defer ch.Close()
		for {
			select {
			case <-ctx.Done():
				c.log.Info().Msg("consumer stopping")
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}
				c.handle(ctx, msg)
			}
		}
	}()

	c.log.Info().Strs("routing_keys", routingKeys).Msg("payment consumer started")
	return nil
}

func (c *CommandConsumer) handle(ctx context.Context, msg amqp.Delivery) {
	env, err := events.Unmarshal(msg.Body)
	if err != nil {
		c.log.Error().Err(err).Msg("decode envelope; sending to DLX")
		_ = msg.Reject(false)
		return
	}
	log := c.log.With().Str("event_id", env.ID).Str("type", env.Type).Logger()
	idemKey := "payment:" + env.ID

	seen, err := c.idem.Seen(ctx, idemKey)
	if err != nil {
		log.Warn().Err(err).Msg("idempotency lookup failed; proceeding anyway")
	}
	if seen {
		log.Debug().Msg("duplicate, skipping")
		_ = msg.Ack(false)
		return
	}

	if err := c.dispatch(ctx, env); err != nil {
		log.Error().Err(err).Msg("handler failed; requeuing")
		_ = msg.Nack(false, true)
		return
	}

	if err := c.idem.Mark(ctx, idemKey); err != nil {
		log.Warn().Err(err).Msg("mark failed; relying on DB unique constraint for next delivery")
	}
	_ = msg.Ack(false)
}

func (c *CommandConsumer) dispatch(ctx context.Context, env events.Envelope) error {
	switch env.Type {
	case "payment.authorize.v1":
		var p events.AuthorizePaymentPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		return c.authorize.Execute(ctx, usecase.AuthorizePaymentInput{
			OrderID:    p.OrderID,
			CustomerID: p.CustomerID,
			Amount:     p.Amount,
		})
	case "payment.release.v1":
		var p events.ReleasePaymentPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		return c.release.Execute(ctx, usecase.ReleasePaymentInput{
			OrderID: p.OrderID,
			Amount:  p.Amount,
		})
	default:
		c.log.Warn().Str("type", env.Type).Msg("unknown command type")
		return nil
	}
}
