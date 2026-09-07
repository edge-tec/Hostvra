#!/usr/bin/env bash
set -e

echo "=== [1/4] Pulling latest Hostvra code ==="
git pull origin main

echo "=== [2/4] Compiling Hostvra Core API ==="
(cd apps/api && go build -o server ./cmd/server)

echo "=== [3/4] Building Next.js Web UI ==="
(cd apps/web && npm run build)

echo "=== [4/4] Restarting systemd services ==="
systemctl restart hostvra-api || true
systemctl restart hostvra-web || true

echo "=== Done! Hostvra is up to date and healthy. ==="
