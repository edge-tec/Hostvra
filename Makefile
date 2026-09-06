.PHONY: all build test lint run-api run-web docker-up docker-down migrate help

all: build

help:
	@echo "Hostvra Monorepo Build Targets:"
	@echo "  make docker-up   - Start PostgreSQL and Redis local containers"
	@echo "  make docker-down - Stop local containers"
	@echo "  make run-api     - Run Go API server locally"
	@echo "  make run-web     - Run Next.js Web UI locally"
	@echo "  make test        - Run backend and frontend test suites"
	@echo "  make build       - Compile Go API and Next.js production bundles"

docker-up:
	docker compose up -d

docker-down:
	docker compose down

run-api:
	cd apps/api && go run cmd/server/main.go

run-web:
	cd apps/web && npm run dev

test: test-api test-web

test-api:
	cd apps/api && go test -v -race ./...

test-web:
	cd apps/web && npm test

build: build-api build-web

build-api:
	cd apps/api && CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/hostvra-api cmd/server/main.go

build-web:
	cd apps/web && npm run build
