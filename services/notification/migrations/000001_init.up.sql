-- Notifications service is end-of-line: it consumes events but doesn't
-- publish, so there's no outbox table here. We only record successful
-- sends; failed sends are returned to the consumer for requeue and don't
-- leave a row behind.
--
-- UNIQUE (order_id, type, channel) is the idempotency boundary: once a
-- notification has been sent for a given order on a given channel for a
-- given event type, we won't send it again.
CREATE TABLE IF NOT EXISTS notifications (
    id          UUID PRIMARY KEY,
    order_id    UUID        NOT NULL,
    customer_id TEXT        NOT NULL,
    type        TEXT        NOT NULL,
    channel     TEXT        NOT NULL,
    sent_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (order_id, type, channel)
);
CREATE INDEX IF NOT EXISTS idx_notifications_order_id ON notifications(order_id);
