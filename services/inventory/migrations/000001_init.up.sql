-- One row per SKU. version + the CHECK constraint together let us do the
-- decrement with a conditional UPDATE that's safe under contention:
--   UPDATE stocks SET quantity = quantity - $qty
--    WHERE sku = $sku AND quantity >= $qty
-- Zero rows affected => insufficient stock or unknown SKU. Caller treats
-- both as a domain failure; the CHECK is just defense in depth.
CREATE TABLE IF NOT EXISTS stocks (
    sku       TEXT    PRIMARY KEY,
    quantity  INT     NOT NULL CHECK (quantity >= 0),
    version   INT     NOT NULL DEFAULT 1
);

-- One reservation per order. UNIQUE(order_id) is the authoritative
-- idempotency guarantee: even if Redis fast-skip misses, a duplicate
-- reserve command can't double-decrement stock.
CREATE TABLE IF NOT EXISTS reservations (
    id          UUID PRIMARY KEY,
    order_id    UUID UNIQUE NOT NULL,
    status      TEXT        NOT NULL,
    reason      TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    version     INT         NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS reservation_items (
    id              BIGSERIAL PRIMARY KEY,
    reservation_id  UUID NOT NULL REFERENCES reservations(id) ON DELETE CASCADE,
    sku             TEXT NOT NULL,
    quantity        INT  NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_reservation_items_reservation_id
    ON reservation_items(reservation_id);

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

-- Seed sample stock so the demo flow works out of the box. Use sku "Z"
-- in a curl to deliberately exercise the failure path.
INSERT INTO stocks (sku, quantity) VALUES
    ('A', 100),
    ('B', 50),
    ('C', 25),
    ('D', 10)
ON CONFLICT (sku) DO NOTHING;
