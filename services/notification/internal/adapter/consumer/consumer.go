// Package consumer is the driving adapter that subscribes to RabbitMQ for
// terminal order events (order.confirmed, order.canceled) and dispatches
// to SendNotification. Post-mark idempotency from pkg/idempotency.
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
	"github.com/lilik-setyawan/orderflow/services/notification/internal/app/usecase"
	"github.com/lilik-setyawan/orderflow/services/notification/internal/domain"
)

const (
	queueName    = "notification.events"
	consumerTag  = "notification-svc"
	prefetchSize = 32
)

var routingKeys = []string{
	"order.confirmed.v1",
	"order.canceled.v1",
}

type EventConsumer struct {
	conn *rabbitmq.Connection
	send *usecase.SendNotification
	idem *idempotency.Cache
	log  zerolog.Logger
}

func New(conn *rabbitmq.Connection, send *usecase.SendNotification, idem *idempotency.Cache, log zerolog.Logger) *EventConsumer {
	return &EventConsumer{
		conn: conn,
		send: send,
		idem: idem,
		log:  log.With().Str("component", "notification-consumer").Logger(),
	}
}

func (c *EventConsumer) Start(ctx context.Context) error {
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

	c.log.Info().Strs("routing_keys", routingKeys).Msg("notification consumer started")
	return nil
}

func (c *EventConsumer) handle(ctx context.Context, msg amqp.Delivery) {
	env, err := events.Unmarshal(msg.Body)
	if err != nil {
		c.log.Error().Err(err).Msg("decode envelope; sending to DLX")
		_ = msg.Reject(false)
		return
	}
	log := c.log.With().Str("event_id", env.ID).Str("type", env.Type).Logger()
	idemKey := "notification:" + env.ID

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

func (c *EventConsumer) dispatch(ctx context.Context, env events.Envelope) error {
	switch env.Type {
	case "order.confirmed.v1":
		var p events.OrderConfirmedPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		return c.send.Execute(ctx, usecase.SendNotificationInput{
			OrderID:    p.OrderID,
			CustomerID: p.CustomerID,
			Type:       domain.TypeOrderConfirmed,
		})
	case "order.canceled.v1":
		var p events.OrderCanceledPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		return c.send.Execute(ctx, usecase.SendNotificationInput{
			OrderID:    p.OrderID,
			CustomerID: p.CustomerID,
			Type:       domain.TypeOrderCanceled,
			Reason:     p.Reason,
		})
	default:
		c.log.Warn().Str("type", env.Type).Msg("unknown event type")
		return nil
	}
}
