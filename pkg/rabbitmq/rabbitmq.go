// Package rabbitmq wraps amqp091-go with the small set of conventions
// OrderFlow relies on: a single durable topic exchange, durable quorum
// queues, dead-letter routing, and publisher confirms.
//
// Topology (declared idempotently on Connect):
//
//	exchange:  orderflow            (topic, durable)
//	exchange:  orderflow.dlx        (topic, durable)  ← dead letters land here
//
// Each consumer service declares its own queue and binds it to the routing
// keys it cares about; see DeclareQueue.
package rabbitmq

import (
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	Exchange    = "orderflow"
	ExchangeDLX = "orderflow.dlx"
)

type Connection struct {
	conn *amqp.Connection
}

func Connect(ctx context.Context, url string) (*Connection, error) {
	conn, err := amqp.DialConfig(url, amqp.Config{
		Heartbeat: 10 * time.Second,
		Locale:    "en_US",
	})
	if err != nil {
		return nil, fmt.Errorf("amqp dial: %w", err)
	}

	// Declare topology once on a throwaway channel.
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open channel: %w", err)
	}
	defer ch.Close()

	for _, ex := range []string{Exchange, ExchangeDLX} {
		if err := ch.ExchangeDeclare(ex, "topic", true, false, false, false, nil); err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("declare exchange %s: %w", ex, err)
		}
	}

	return &Connection{conn: conn}, nil
}

func (c *Connection) Raw() *amqp.Connection { return c.conn }

func (c *Connection) Close() error {
	if c.conn == nil || c.conn.IsClosed() {
		return nil
	}
	return c.conn.Close()
}

// DeclareQueue creates a durable quorum queue with a DLX bound to ExchangeDLX,
// and binds it to each routing key. Idempotent; safe to call on every boot.
func (c *Connection) DeclareQueue(name string, routingKeys []string) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	args := amqp.Table{
		"x-queue-type":           "quorum",
		"x-dead-letter-exchange": ExchangeDLX,
	}
	if _, err := ch.QueueDeclare(name, true, false, false, false, args); err != nil {
		return fmt.Errorf("declare queue %s: %w", name, err)
	}
	for _, rk := range routingKeys {
		if err := ch.QueueBind(name, rk, Exchange, false, nil); err != nil {
			return fmt.Errorf("bind queue %s -> %s: %w", name, rk, err)
		}
	}

	// Mirror queue on the DLX so failed messages are observable + replay-able.
	dlqName := name + ".dlq"
	if _, err := ch.QueueDeclare(dlqName, true, false, false, false, amqp.Table{
		"x-queue-type": "quorum",
	}); err != nil {
		return fmt.Errorf("declare dlq %s: %w", dlqName, err)
	}
	for _, rk := range routingKeys {
		if err := ch.QueueBind(dlqName, rk, ExchangeDLX, false, nil); err != nil {
			return fmt.Errorf("bind dlq %s -> %s: %w", dlqName, rk, err)
		}
	}
	return nil
}
