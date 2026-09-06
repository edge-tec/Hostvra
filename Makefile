.PHONY: all build test lint run-api run-web run-agent docker-up docker-down migrate help

all: build

help:
	@echo "Hostvra Monorepo Build Targets:"
	@echo "  make docker-up    - Start PostgreSQL and Redis local containers"
	@echo "  make docker-down  - Stop local containers"
	@echo "  make run-api      - Run Go API server locally"
	@echo "  make run-web      - Run Next.js Web UI locally"
	@echo "  make run-agent    - Run Hostvra Agent daemon locally"
	@echo "  make test         - Run all test suites (API, Agent, Web)"
	@echo "  make build        - Compile Go API, Agent, and Next.js bundles"

docker-up:
	docker compose up -d

docker-down:
	docker compose down

run-api:
	cd apps/api && go run cmd/server/main.go

run-web:
	cd apps/web && npm run dev

run-agent:
	cd apps/agent && go run cmd/agent/main.go --daemon

test: test-api test-agent test-web

test-api:
	cd apps/api && go test -v -race ./...

test-agent:
	cd apps/agent && go test -v ./...

test-web:
	cd apps/web && npm run build

build: build-api build-agent build-web

build-api:
	cd apps/api && CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/hostvra-api cmd/server/main.go

build-agent:
	cd apps/agent && CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/hostvra-agent cmd/agent/main.go
	cd apps/agent && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/hostvra-agent-linux-amd64 cmd/agent/main.go

build-web:
	cd apps/web && npm run build
