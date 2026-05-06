// Package consumer is the driving adapter that subscribes to RabbitMQ saga
// reply events and invokes the matching saga use case. Idempotency, decoding,
// and ack/nack policy live here — the use case stays transport-agnostic.
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
	"github.com/lilik-setyawan/orderflow/services/order/internal/app/usecase"
)

const (
	queueName    = "order.saga"
	consumerTag  = "order-saga"
	prefetchSize = 32
)

var routingKeys = []string{
	"payment.authorized.v1",
	"payment.failed.v1",
	"payment.released.v1",
	"inventory.reserved.v1",
	"inventory.failed.v1",
	"inventory.released.v1",
}

type SagaConsumer struct {
	conn *rabbitmq.Connection
	saga *usecase.Saga
	idem *idempotency.Cache
	log  zerolog.Logger
}

func NewSaga(conn *rabbitmq.Connection, saga *usecase.Saga, idem *idempotency.Cache, log zerolog.Logger) *SagaConsumer {
	return &SagaConsumer{
		conn: conn,
		saga: saga,
		idem: idem,
		log:  log.With().Str("component", "saga-consumer").Logger(),
	}
}

func (c *SagaConsumer) Start(ctx context.Context) error {
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

	c.log.Info().Strs("routing_keys", routingKeys).Msg("saga consumer started")
	return nil
}

// handle uses the post-mark idempotency pattern: skip fast if already marked,
// otherwise run handler then mark on success. Failed handlers requeue —
// retries re-run the use case, which is itself idempotent thanks to state
// guards in the saga.
func (c *SagaConsumer) handle(ctx context.Context, msg amqp.Delivery) {
	env, err := events.Unmarshal(msg.Body)
	if err != nil {
		c.log.Error().Err(err).Msg("decode envelope; sending to DLX")
		_ = msg.Reject(false)
		return
	}
	log := c.log.With().Str("event_id", env.ID).Str("type", env.Type).Logger()
	idemKey := "saga:" + env.ID

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
		log.Warn().Err(err).Msg("mark failed; relying on state guards for next delivery")
	}
	_ = msg.Ack(false)
}

func (c *SagaConsumer) dispatch(ctx context.Context, env events.Envelope) error {
	switch env.Type {
	case "payment.authorized.v1":
		var p events.PaymentAuthorizedPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		return c.saga.OnPaymentAuthorized(ctx, p)
	case "payment.failed.v1":
		var p events.PaymentFailedPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		return c.saga.OnPaymentFailed(ctx, p)
	case "inventory.reserved.v1":
		var p events.InventoryReservedPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		return c.saga.OnInventoryReserved(ctx, p)
	case "inventory.failed.v1":
		var p events.InventoryFailedPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		return c.saga.OnInventoryFailed(ctx, p)
	case "payment.released.v1", "inventory.released.v1":
		c.log.Info().Str("type", env.Type).Msg("compensation observed")
		return nil
	default:
		c.log.Warn().Str("type", env.Type).Msg("unknown event type")
		return nil
	}
}
