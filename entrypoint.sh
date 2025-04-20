#!/usr/bin/env bash
set -e

# Start Meilisearch in the background
meilisearch --http-addr 0.0.0.0:7700 --master-key "${MEILI_MASTER_KEY:-masterKey}" &
MEILI_PID=$!

# Poll /health until status is "available" (up to CHECK_RETRIES, default 30)
CHECK_RETRIES="${CHECK_RETRIES:-30}"
echo "Waiting for Meilisearch (max ${CHECK_RETRIES} retries)..."
count=0
until curl -s http://localhost:7700/health | grep -q '"available"'; do
  count=$((count + 1))
  if [ "$count" -ge "$CHECK_RETRIES" ]; then
    echo "ERROR: Meilisearch did not become available after ${CHECK_RETRIES} attempts." >&2
    kill "$MEILI_PID"
    exit 1
  fi
  echo "Retry #${count}..."
  sleep 1
done
echo "Meilisearch is available after ${count} retries."

# Auto‑index Markdown files in each host‑mounted DOCUMENT_DIRS
if [ -n "$DOCUMENT_DIRS" ]; then
  IFS=',' read -ra DIRS <<< "$DOCUMENT_DIRS"
  for DIR in "${DIRS[@]}"; do
    echo "Indexing Markdown files in: $DIR"
    find "$DIR" -type f -name '*.md' -print0 | while IFS= read -r -d '' file; do
      id=$(basename "$file" .md)
      content=$(jq -Rs . < "$file")
      curl -s -X POST "http://localhost:7700/indexes/documents/documents" \
           -H "Content-Type: application/json" \
           --data "[{\"id\":\"$id\",\"content\":$content}]"
    done
  done
fi

# Finally, launch the MCP server
exec meilisearch-mcp