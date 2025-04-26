#!/usr/bin/env bash
set -e
until curl -sf "${MEILI_HTTP_ADDR:-http://localhost:7700}/health" | grep -q available; do
  echo "🌱 waiting for Meilisearch..."
  sleep 1
done
