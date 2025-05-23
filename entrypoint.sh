#!/usr/bin/env bash
set -e

# Set default master key if not provided
: ${MEILI_MASTER_KEY:="masterKey"}
# Export MEILI_MASTER_KEY as an environment variable
export MEILI_MASTER_KEY

# Start Meilisearch in the background
echo "Starting Meilisearch server..." >&2
/bin/meilisearch --master-key="$MEILI_MASTER_KEY" &
MEILI_PID=$!

# Wait for Meilisearch to be ready
echo "Waiting for Meilisearch to be ready..." >&2
MAX_RETRIES=${CHECK_RETRIES:-30}
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
  if curl -s -f "http://localhost:7700/health" | grep -q '{"status":"available"}'; then
    echo "Meilisearch is ready!" >&2
    break
  fi
  echo "Waiting for Meilisearch to be ready... ($((RETRY_COUNT+1))/$MAX_RETRIES)" >&2
  sleep 1
  RETRY_COUNT=$((RETRY_COUNT+1))
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
  echo "Meilisearch failed to start within the timeout period." >&2
  exit 1
fi

# Auto‑index Markdown files in each host‑mounted DOCUMENT_DIRS
if [ -n "$DOCUMENT_DIRS" ]; then
  IFS=',' read -ra DIRS <<< "$DOCUMENT_DIRS"
  for DIR in "${DIRS[@]}"; do
    echo "Indexing Markdown files in: $DIR" >&2
    find "$DIR" -type f -name '*.md' -print0 | while IFS= read -r -d '' file; do
      id=$(basename "$file" .md)
      content=$(jq -Rs . < "$file")
      curl -s -X POST "http://localhost:7700/indexes/documents/documents" \
           -H "Content-Type: application/json" \
           -H "Authorization: Bearer $MEILI_MASTER_KEY" \
           --data "[{\"id\":\"$id\",\"content\":$content}]"
    done
  done
fi

# Launch the MCP server
echo "Starting MCP server..." >&2

cd /app/meilisearch-mcp

uv venv

# Activate the virtual environment
source .venv/bin/activate

uv pip install -e .

# Run the MCP server
exec python -m meilisearch_mcp