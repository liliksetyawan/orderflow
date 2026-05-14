# OrderFlow Agent Instructions

Use these instructions when working in this repository.

## Project Shape

- `orderflow` is a Go microservices project that demonstrates saga orchestration, transactional outbox, idempotent consumers, RabbitMQ messaging, PostgreSQL schema-per-service storage, Redis idempotency cache, and OpenTelemetry wiring.
- The backend module is `github.com/liliksetyawan/orderflow`.
- The frontend lives in `web` and uses React, Vite, TypeScript, RTK Query, React Router, Tailwind CSS, and shadcn-style local UI components.

## Backend Architecture

- Keep the existing hexagonal architecture:
  - `internal/domain`: entities, state transitions, validation, and domain errors only.
  - `internal/app/port`: interfaces owned by the application layer.
  - `internal/app/usecase`: orchestration over domain + ports.
  - `internal/adapter/*`: infrastructure implementations such as HTTP, RabbitMQ consumers, PostgreSQL repositories, ID generation, gateways, and notifiers.
  - `cmd/server/main.go`: composition root only.
- Do not make domain packages import infrastructure, adapters, config, logging, database, messaging, or HTTP packages.
- Prefer adding behavior to use cases and domain methods before reaching into adapters.
- If a port changes, regenerate mocks with `make mocks`.

## Services

- `services/order`: client-facing REST API and saga orchestrator.
  - Creates orders.
  - Emits `order.created.v1` and `payment.authorize.v1`.
  - Reacts to payment and inventory result events.
- `services/payment`: AMQP-driven payment processor.
  - Handles `payment.authorize.v1` and `payment.release.v1`.
  - Emits payment result events through outbox.
- `services/inventory`: AMQP-driven stock reservation service.
  - Handles `inventory.reserve.v1` and `inventory.release.v1`.
  - Uses conditional stock updates to avoid oversell.
- `services/notification`: terminal consumer.
  - Handles confirmed/canceled order events.
  - Records successful sends and does not publish outbox events.

## Shared Packages

- `pkg/events`: cross-service event envelope and payload contracts. Keep event type strings stable unless all producers and consumers are updated together.
- `pkg/outbox`: transactional outbox writer and dispatcher. Services that publish events should write business state and outbox rows in the same transaction.
- `pkg/rabbitmq`: RabbitMQ topology and publisher confirms.
- `pkg/idempotency`: Redis-based fast-path consumer idempotency.
- `pkg/pgx`: PostgreSQL connection helper.
- `pkg/dbmigrate`: embedded SQL migration runner.
- `pkg/observability`: zerolog and OpenTelemetry setup.

## Messaging And Idempotency

- Assume RabbitMQ delivery is at-least-once.
- Consumers must be idempotent.
- Preserve the existing three-layer strategy:
  - Redis fast-skip by message/event ID.
  - Database uniqueness constraints for authoritative deduplication.
  - Domain state guards for duplicate or late saga events.
- Do not publish directly from use cases or repositories. Persist domain state plus outbox records transactionally, then let the dispatcher publish.
- When adding a new event:
  - Add the event type constant and payload in `pkg/events`.
  - Map domain event types to routing keys in the relevant adapter.
  - Update consumers and tests for all affected services.

## Database Rules

- Each service owns its schema:
  - `order_svc`
  - `payment_svc`
  - `inventory_svc`
  - `notification_svc`
- Keep migrations embedded under each service's `migrations/` directory.
- For state updates, preserve optimistic concurrency patterns using `version` columns.
- For inventory, preserve conditional stock decrement semantics. Unknown or insufficient SKU should become an inventory failure, not oversell.

## Frontend Rules

- Keep frontend API contracts aligned with the Go HTTP DTOs in `web/src/lib/types.ts`.
- Use RTK Query for server data access.
- Existing API slices:
  - `web/src/features/orders/ordersApi.ts`
  - `web/src/features/health/healthApi.ts`
- Prefer existing components under `web/src/components`.
- Do not introduce a new UI framework unless explicitly requested.
- Keep routes in `web/src/App.tsx` and page-level workflows in `web/src/pages`.

## Local Commands

From the repository root:

```bash
make up
make run-order
make run-payment
make run-inventory
make run-notification
make test
go test ./...
```

From `web`:

```bash
npm run build
npm run dev
```

## Verification Expectations

- For backend changes, run `go test ./...`.
- For frontend changes, run `npm run build` from `web`.
- For changes that alter ports, run `make mocks` and include generated mock updates.
- For message contract changes, add or update tests around event mapping and affected use cases.
- For schema changes, add forward and rollback migrations.

## Current Known Notes

- Unit tests pass with `go test ./...`.
- Frontend production build passes with `npm run build`.
- OpenTelemetry setup exists, but README marks full trace instrumentation as pending.
- Integration tests with PostgreSQL/RabbitMQ are not yet implemented.
