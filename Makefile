.PHONY: up down build run test tidy logs

up:
	docker compose up -d --build

down:
	docker compose down -v

build:
	go build -o bin/gateway ./cmd/gateway

run:
	go run ./cmd/gateway

test:
	go test ./...

tidy:
	go mod tidy

logs:
	docker compose logs -f gateway
