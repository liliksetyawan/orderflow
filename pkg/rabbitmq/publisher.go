package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Publisher publishes JSON messages to the orderflow exchange with publisher
// confirms enabled. Each call blocks until the broker confirms or the context
// expires — that guarantee is what the outbox dispatcher relies on to know
// it's safe to mark a row sent.
type Publisher struct {
	mu       sync.Mutex
	conn     *Connection
	exchange string
	ch       *amqp.Channel
}

func NewPublisher(conn *Connection) (*Publisher, error) {
	p := &Publisher{conn: conn, exchange: Exchange}
	if err := p.openChannel(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Publisher) openChannel() error {
	ch, err := p.conn.conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	if err := ch.Confirm(false); err != nil {
		_ = ch.Close()
		return fmt.Errorf("enable confirms: %w", err)
	}
	p.ch = ch
	return nil
}

// Publish sends body to routingKey with msgID set as MessageId (used by
// consumers as the idempotency key). Blocks until confirmed.
func (p *Publisher) Publish(ctx context.Context, routingKey, msgID string, body []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ch == nil || p.ch.IsClosed() {
		if err := p.openChannel(); err != nil {
			return err
		}
	}

	conf, err := p.ch.PublishWithDeferredConfirmWithContext(ctx, p.exchange, routingKey, true, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    msgID,
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	ack, err := conf.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("await confirm: %w", err)
	}
	if !ack {
		return errors.New("broker nacked publish")
	}
	return nil
}

func (p *Publisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ch != nil && !p.ch.IsClosed() {
		return p.ch.Close()
	}
	return nil
}
