// Package consumer is the driving adapter that subscribes to RabbitMQ for
// inventory.* commands and dispatches to the matching use case. Post-mark
// idempotency from pkg/idempotency.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"

	"github.com/liliksetyawan/orderflow/pkg/events"
	"github.com/liliksetyawan/orderflow/pkg/idempotency"
	"github.com/liliksetyawan/orderflow/pkg/rabbitmq"
	"github.com/liliksetyawan/orderflow/services/inventory/internal/app/usecase"
	"github.com/liliksetyawan/orderflow/services/inventory/internal/domain"
)

const (
	queueName    = "inventory.commands"
	consumerTag  = "inventory-svc"
	prefetchSize = 32
)

var routingKeys = []string{
	"inventory.reserve.v1",
	"inventory.release.v1",
}

type CommandConsumer struct {
	conn    *rabbitmq.Connection
	reserve *usecase.ReserveInventory
	release *usecase.ReleaseInventory
	idem    *idempotency.Cache
	log     zerolog.Logger
}

func New(conn *rabbitmq.Connection, reserve *usecase.ReserveInventory, release *usecase.ReleaseInventory, idem *idempotency.Cache, log zerolog.Logger) *CommandConsumer {
	return &CommandConsumer{
		conn:    conn,
		reserve: reserve,
		release: release,
		idem:    idem,
		log:     log.With().Str("component", "inventory-consumer").Logger(),
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

	c.log.Info().Strs("routing_keys", routingKeys).Msg("inventory consumer started")
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
	idemKey := "inventory:" + env.ID

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
	case "inventory.reserve.v1":
		var p events.ReserveInventoryPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		items := make([]domain.Item, len(p.Items))
		for i, it := range p.Items {
			items[i] = domain.Item{SKU: it.SKU, Quantity: it.Quantity}
		}
		return c.reserve.Execute(ctx, usecase.ReserveInventoryInput{
			OrderID: p.OrderID,
			Items:   items,
		})
	case "inventory.release.v1":
		var p events.ReleaseInventoryPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		return c.release.Execute(ctx, usecase.ReleaseInventoryInput{
			OrderID: p.OrderID,
		})
	default:
		c.log.Warn().Str("type", env.Type).Msg("unknown command type")
		return nil
	}
}
