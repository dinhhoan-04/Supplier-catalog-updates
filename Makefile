.PHONY: up down build test test-unit restart logs clean

up:
	docker compose up -d

build:
	docker compose up --build -d

down:
	docker compose down

restart:
	docker compose restart

logs:
	docker compose logs -f

test:
	docker run --rm -v "$(PWD)/backend:/app" -w /app golang:1.23-alpine go test -v ./...

clean:
	docker compose down -v
