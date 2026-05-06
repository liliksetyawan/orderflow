-- One payment per order. The UNIQUE constraint on order_id is the
-- authoritative idempotency guarantee: even if the broker delivers the
-- same authorize command twice and Redis fast-skip misses, the second
-- INSERT fails with 23505 and the use case treats it as success.
CREATE TABLE IF NOT EXISTS payments (
    id           UUID PRIMARY KEY,
    order_id     UUID UNIQUE NOT NULL,
    customer_id  TEXT        NOT NULL,
    amount       BIGINT      NOT NULL,
    status       TEXT        NOT NULL,
    provider_ref TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    version      INT         NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS outbox (
    id           UUID PRIMARY KEY,
    routing_key  TEXT        NOT NULL,
    payload      JSONB       NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at      TIMESTAMPTZ,
    attempts     INT         NOT NULL DEFAULT 0,
    last_error   TEXT
);
CREATE INDEX IF NOT EXISTS idx_outbox_unsent
    ON outbox (created_at) WHERE sent_at IS NULL;
