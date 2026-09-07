#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=== [1/5] Pulling latest Hostvra code ==="
git pull origin main

echo "=== [2/5] Compiling Hostvra Core API & CLI ==="
mkdir -p bin
(cd apps/api && go build -o server ./cmd/server)
cp -f apps/api/server bin/hostvra-api 2>/dev/null || true

if [[ -w "/usr/local/bin" ]]; then
    cp -f apps/api/server /usr/local/bin/hostvra-api 2>/dev/null || true
    chmod +x /usr/local/bin/hostvra-api 2>/dev/null || true
fi

echo "=== [3/5] Compiling Hostvra Node Agent ==="
if [[ -d "apps/agent" ]]; then
    (cd apps/agent && go build -o agent ./cmd/agent)
    cp -f apps/agent/agent bin/hostvra-agent 2>/dev/null || true
    if [[ -w "/usr/local/bin" ]]; then
        cp -f apps/agent/agent /usr/local/bin/hostvra-agent 2>/dev/null || true
        chmod +x /usr/local/bin/hostvra-agent 2>/dev/null || true
    fi
fi

echo "=== [4/5] Building Next.js Web UI ==="
(cd apps/web && npm run build)

echo "=== [5/5] Scheduling Hostvra services reload ==="
if command -v systemctl &>/dev/null; then
    (
        sleep 2
        systemctl restart hostvra-api 2>/dev/null || true
        systemctl restart hostvra-web 2>/dev/null || true
        systemctl restart hostvra-agent 2>/dev/null || true
    ) >/dev/null 2>&1 &
fi

echo "=== Done! Hostvra is successfully updated. Services will reload in 2 seconds. ==="
