.PHONY: all build test lint run-api run-web run-agent docker-up docker-down migrate help compile-linux

all: build

help:
	@echo "Hostvra Monorepo Build Targets:"
	@echo "  make docker-up       - Start PostgreSQL and Redis local containers"
	@echo "  make docker-down     - Stop local containers"
	@echo "  make run-api         - Run Go API server locally (port 8080)"
	@echo "  make run-web         - Run Next.js Web UI locally (port 3000)"
	@echo "  make run-agent       - Run Hostvra Agent daemon locally"
	@echo "  make test            - Run all test suites with race detector"
	@echo "  make build           - Compile native Go API, Agent, and Next.js bundles"
	@echo "  make compile-linux   - Cross-compile static Linux amd64 and arm64 binaries"

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
	cd apps/agent && go test -v -race ./...

test-web:
	cd apps/web && npm run build

build: build-api build-agent build-web

build-api:
	@mkdir -p bin
	cd apps/api && CGO_ENABLED=0 go build -ldflags="-s -w" -o ../../bin/hostvra-api cmd/server/main.go

build-agent:
	@mkdir -p bin
	cd apps/agent && CGO_ENABLED=0 go build -ldflags="-s -w" -o ../../bin/hostvra-agent cmd/agent/main.go

build-web:
	cd apps/web && npm run build

compile-linux:
	@mkdir -p bin
	cd apps/api && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ../../bin/hostvra-api-linux-amd64 cmd/server/main.go
	cd apps/api && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o ../../bin/hostvra-api-linux-arm64 cmd/server/main.go
	cd apps/agent && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ../../bin/hostvra-agent-linux-amd64 cmd/agent/main.go
	cd apps/agent && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o ../../bin/hostvra-agent-linux-arm64 cmd/agent/main.go
	@ls -lh bin/
