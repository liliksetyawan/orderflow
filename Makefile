.PHONY: help up down logs ps psql nats-stream \
        tidy build test test-cover lint \
        run-order run-payment run-inventory run-notification \
        migrate clean

SHELL := /bin/bash
SERVICES := order payment inventory notification

## help: list common targets
help:
	@grep -E '^##' Makefile | sed -e 's/## //'

## up: start postgres, rabbitmq, redis, jaeger
up:
	docker compose up -d
	@echo "Waiting for services..."
	@docker compose ps

## down: stop infra
down:
	docker compose down

## logs: tail infra logs
logs:
	docker compose logs -f

## ps: list compose services
ps:
	docker compose ps

## psql: psql shell into orderflow db
psql:
	docker compose exec postgres psql -U orderflow -d orderflow

## tidy: go mod tidy
tidy:
	go mod tidy

## mocks: regenerate gomock mocks for all port packages
mocks:
	go generate ./...

## build: build all service binaries to bin/
build:
	@mkdir -p bin
	@for s in $(SERVICES); do \
		echo ">> building $$s"; \
		go build -o bin/$$s ./services/$$s/cmd/server || exit 1; \
	done

## test: run all tests
test:
	go test ./...

## test-cover: run tests with coverage
test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

## lint: golangci-lint (must be installed)
lint:
	golangci-lint run ./...

## run-order: run order service locally
run-order:
	go run ./services/order/cmd/server

## run-payment: run payment service locally
run-payment:
	go run ./services/payment/cmd/server

## run-inventory: run inventory service locally
run-inventory:
	go run ./services/inventory/cmd/server

## run-notification: run notification service locally
run-notification:
	go run ./services/notification/cmd/server

## clean: remove build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html
