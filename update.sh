#!/usr/bin/env bash
set -e

echo "=== [1/4] Pulling latest Hostvra code ==="
git pull origin main

echo "=== [2/4] Compiling Hostvra Core API ==="
(cd apps/api && go build -o server ./cmd/server)

echo "=== [3/4] Building Next.js Web UI ==="
(cd apps/web && npm run build)

echo "=== [4/4] Scheduling Hostvra service reload ==="
(sleep 2 && systemctl restart hostvra-api && systemctl restart hostvra-web) >/dev/null 2>&1 &

echo "=== Done! Hostvra is successfully updated. Services will reload in 2 seconds. ==="
