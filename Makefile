SHELL := /bin/bash

TEST_DB_URL=postgres://test:test@127.0.0.1:55432/cahaya_test?sslmode=disable

.PHONY: test-db-up test-db-down test-db-reset test-migrate-up test-integration test

test-db-up:
	docker compose -f docker-compose.test.yml up -d
	@echo "Waiting for postgres..."
	@until docker exec cahaya_pg_test pg_isready -U test -d cahaya_test >/dev/null 2>&1; do sleep 1; done
	@echo "Postgres is ready."

test-db-down:
	docker compose -f docker-compose.test.yml down -v

test-db-reset: test-db-down test-db-up

test-migrate-up:
	DATABASE_URL="$(TEST_DB_URL)" goose -dir ./migrations postgres "$(TEST_DB_URL)" up

test-integration: test-db-up test-migrate-up
	DATABASE_URL="$(TEST_DB_URL)" go test ./... -count=1

test:
	go test ./... -count=1
