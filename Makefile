.PHONY: build test lint run clean cover bench proto docker-up docker-down migrate

BINARY=omnitest
BUILD_DIR=bin

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/omnitest

test:
	go test -v -race ./...

lint:
	golangci-lint run ./...

run: build
	./$(BUILD_DIR)/$(BINARY) run testdata/sample.yaml

cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

bench:
	go test -bench=. -benchmem ./...

clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html

# Cycle 2: Distributed Architecture

proto:
	protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/omnitest/v1/agent.proto

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

migrate:
	@echo "Applying migrations to PostgreSQL..."
	PGPASSWORD=omnitest psql -h localhost -U omnitest -d omnitest -f migrations/001_init.sql
