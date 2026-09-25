---
name: go-backend
description: |
  Go backend development guide for Hostvra API and Agent.
  Covers the Chi router pattern, handler structure, store interface,
  RBAC middleware, audit logging, and testing conventions.
---

# Hostvra Go Backend Reference

## Module Structure

### API (hostvra/api — go 1.26)
```
apps/api/
├── cmd/server/main.go     — entry point, router setup, handler wiring
├── cmd/hostvra/main.go    — CLI tool
└── internal/
    ├── handlers/          — HTTP handlers (62 files)
    ├── store/             — database layer
    ├── auth/              — JWT + Argon2id
    ├── rbac/              — permission system
    ├── audit/             — audit logger
    ├── config/            — env-based config
    ├── update/            — live update orchestrator
    └── ...
```

### Agent (hostvra/agent — go 1.22)
```
apps/agent/
├── cmd/agent/main.go      — enrollment + telemetry daemon
├── internal/              — client, collector, config, osadapter
└── pkg/                   — 17 infrastructure capability packages
```

## Handler Pattern

Every handler follows this exact structure:

```go
type FooHandler struct {
    cfg   *config.Config
    store store.Store
    audit *audit.Logger
    // optional: domain-specific manager from hostvra/agent/pkg/
}

func NewFooHandler(cfg *config.Config, s store.Store, a *audit.Logger) *FooHandler {
    return &FooHandler{cfg: cfg, store: s, audit: a}
}

func (h *FooHandler) DoThing(w http.ResponseWriter, r *http.Request) {
    // 1. Get auth claims
    claims, ok := auth.GetClaims(r.Context())
    if !ok {
        response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "...", nil, "")
        return
    }

    // 2. Decode + validate request body
    var req FooRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.Error(w, http.StatusBadRequest, "INVALID_BODY", err.Error(), nil, "")
        return
    }

    // 3. Business logic via store or agent/pkg
    result, err := h.store.GetFoo(r.Context(), claims.OrganizationID)
    if err != nil {
        response.Error(w, http.StatusInternalServerError, "FOO_ERROR", err.Error(), nil, "")
        return
    }

    // 4. Audit log
    h.audit.Log(r.Context(), r, "foo.do_thing", "foo", result.ID.String(), "success", "", map[string]interface{}{})

    // 5. Respond
    response.JSON(w, http.StatusOK, result, nil)
}
```

## RBAC Middleware Usage

```go
r.With(rbac.RequirePermission(rbac.PermWebsitesCreate)).Post("/", handler.Create)
```

Roles and permissions are defined in `internal/rbac/rbac.go`.
Never hardcode permission strings — always use the `rbac.Perm*` constants.

## Store Interface Pattern

The `Store` interface in `internal/store/store.go` is implemented by:
- `PostgresStore` — production (database/sql with lib/pq)
- `MemoryStore` — development fallback (sync.Map)

To add a new operation:
1. Add method signature to the `Store` interface
2. Implement in both `PostgresStore` and `MemoryStore`
3. Add corresponding model structs in appropriate `*_models.go` file

## Test Pattern

Handler tests use in-memory store and mock HTTP:

```go
func TestFooHandler_DoThing(t *testing.T) {
    store := store.NewMemoryStore()
    auditLogger := audit.NewLogger(store, slog.Default())
    handler := handlers.NewFooHandler(testConfig(), store, auditLogger)

    // Seed test data in store
    // Create JWT claims context
    // Make HTTP request via httptest
    // Assert response
}
```

Run all API tests:
```bash
cd apps/api && go test -count=1 -v -race ./...
```

Run all Agent tests:
```bash
cd apps/agent && go test -count=1 -v -race ./...
```

Run vet:
```bash
cd apps/api && go vet ./...
cd apps/agent && go vet ./...
```

## Build Commands

```bash
# Development run
cd apps/api && go run cmd/server/main.go

# Production binary
cd apps/api && CGO_ENABLED=0 go build -ldflags="-s -w" -o ../../bin/hostvra-api cmd/server/main.go
```

## Key Rules

1. NEVER use `fmt.Println` — use `slog` structured logging
2. NEVER ignore errors — always handle or propagate
3. NEVER execute shell commands in handlers directly (use agent/pkg/)
4. ALWAYS audit sensitive operations
5. ALWAYS use RBAC middleware on protected routes
6. ALWAYS validate inputs before processing
7. NEVER return raw error messages to clients — wrap them
