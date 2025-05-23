# Meilisearch + MCP Server (Single‑Container)

This repository provides a single Docker container that bundles:

1. Meilisearch (full‑text + vector search engine, with built‑in Japanese tokenizer Lindera).
2. Meilisearch MCP server (`meilisearch-mcp` Python package) for Model Context Protocol integration.

## Features

- Japanese & English Markdown search using Lindera tokenizer.
- Auto‑index up to three host‑mounted directories of `.md` files at container startup.
- Configurable startup behavior via environment variables:
  - `DOCUMENT_DIRS` (comma‑separated host paths, e.g. `/host/one,/host/two,/host/three`)
  - `CHECK_RETRIES` (health‑check polling attempts, default: `30`)
  - `MEILI_MASTER_KEY` (default: `masterKey`)

## MCP Client Configuration Example

If you are using an MCP client that requires a configuration file (e.g., in JSON format) and supports launching the MCP server as a subprocess using Docker, here is an example of how you might configure it:

```json
{
  "mcpServers": {
    "meilisearch-mcp": {
      "command": "docker",
      "args": [
        "run",
        "-i",
        "--rm",
        "-e", "DOCUMENT_DIRS",
        "-v", "/path/to/your/repo/example/spec:/app/example/spec",
        "-v", "/path/to/your/repo/example/guide:/app/example/guide",
        "ghcr.io/upamune/meilisearch-lite-mcp:latest"
      ],
      "env": {
        "DOCUMENT_DIRS": "/app/example/spec,/app/example/guide"
      }
    }
  }
}
```
This example shows how a client could potentially launch the Docker container as a subprocess. Replace `/path/to/your/repo/example/spec` and `/path/to/your/repo/example/guide` with the actual paths on your host machine. The exact configuration format may vary depending on the MCP client you are using.

## Files

- **Dockerfile**
  Single‑stage build: installs runtime deps, clones & installs `meilisearch-mcp`, downloads Meilisearch binary, and sets up entrypoint.

- **entrypoint.sh**
  1. Launches Meilisearch in background.
  2. Polls `/health` until `{"status":"available"}`.
  3. Finds and indexes `.md` files in each `DOCUMENT_DIRS` path.
  4. Starts the MCP server.

## Usage

Run the container using the image from GitHub Container Registry (GHCR):

    docker run --pull always --rm \
      -e DOCUMENT_DIRS="/app/example/spec,/app/example/guide" \
      -e CHECK_RETRIES=30 \
      -v /path/to/your/repo/example/spec:/app/example/spec \
      -v /path/to/your/repo/example/guide:/app/example/guide \
      ghcr.io/upamune/meilisearch-lite-mcp:latest
