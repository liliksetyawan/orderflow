-- One physical postgres, one schema per service. Cheaper than four containers
-- in dev; in prod each service would point at its own database/cluster.
CREATE SCHEMA IF NOT EXISTS order_svc;
CREATE SCHEMA IF NOT EXISTS payment_svc;
CREATE SCHEMA IF NOT EXISTS inventory_svc;
CREATE SCHEMA IF NOT EXISTS notification_svc;
