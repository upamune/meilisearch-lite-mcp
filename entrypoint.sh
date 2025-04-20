#!/usr/bin/env bash
set -e

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
. /opt/venv/bin/activate
exec meilisearch-mcp