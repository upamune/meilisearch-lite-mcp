# Meilisearch + MCP Server (Single‑Container)

This repository provides a single Docker container that bundles:

1. Meilisearch (full‑text + vector search engine, with built‑in Japanese tokenizer Lindera).  
2. Meilisearch MCP server (`meilisearch-mcp` Python package) for Model Context Protocol integration.

## Features

- Japanese & English Markdown search using Lindera tokenizer.  
- Auto‑index up to three host‑mounted directories of `.md` files at container startup.  
- Configurable startup behavior via environment variables:
  - `MEILI_MASTER_KEY` (default: `masterKey`)  
  - `DOCUMENT_DIRS` (comma‑separated host paths, e.g. `/host/one,/host/two,/host/three`)  
  - `CHECK_RETRIES` (health‑check polling attempts, default: `30`)

## Files

- **Dockerfile**  
  Single‑stage build: installs runtime deps, clones & installs `meilisearch-mcp`, downloads Meilisearch binary, and sets up entrypoint.

- **entrypoint.sh**  
  1. Launches Meilisearch in background.  
  2. Polls `/health` until `{"status":"available"}`.  
  3. Finds and indexes `.md` files in each `DOCUMENT_DIRS` path.  
  4. Starts the MCP server.

## Usage

Build the image by running:

    docker build -t meilisearch-mcp:local .

Run the container (example mapping to avoid port conflicts):

    docker run -d \
      --name meili-mcp \
      -p 8777:7700 \      # Host port 8777 → container 7700 (Meilisearch)  
      -p 8300:3000 \      # Host port 8300 → container 3000 (MCP server)  
      -e MEILI_MASTER_KEY=masterKey \
      -e DOCUMENT_DIRS="/home/user/notes/one,/home/user/notes/two,/home/user/notes/three" \
      -e CHECK_RETRIES=30 \
      -v /home/user/notes/one:/docs/one \
      -v /home/user/notes/two:/docs/two \
      -v /home/user/notes/three:/docs/three \
      meilisearch-mcp:local

Access Meilisearch at http://localhost:8777 and the MCP server at http://localhost:8300.