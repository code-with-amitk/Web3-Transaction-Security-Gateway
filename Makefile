.PHONY: up down build run test tidy logs

up:
	docker compose up -d --build

down:
	docker compose down -v

build:
	go build -o bin/gateway ./cmd/gateway

build-client:
	go build -o bin/client ./cmd/client

run:
	go run ./cmd/gateway

test:
	go test ./...

tidy:
	go mod tidy

env:
	@test -f .env || cp .env.example .env
	@echo "Created .env from .env.example (edit as needed)"

logs:
	docker compose logs -f gateway
