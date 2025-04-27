# Meilisearch Hybrid Search MCP & CLI

This repository provides tools for leveraging Meilisearch's hybrid search capabilities (vector + keyword):

1.  **Meilisearch MCP Server:** An MCP (Model Context Protocol) server (`cmd/mcp serve`) that connects to a Meilisearch instance, allowing MCP clients to perform hybrid searches on indexed documents.
2.  **Interactive Search CLI:** A command-line interface (`cmd/mcp search`) for interactively performing hybrid searches directly against Meilisearch.
3.  **Document Processing Utilities:** Go packages (`internal/splitter`, `internal/meilisearchutil`) for splitting documents (especially Markdown) into chunks suitable for embedding and for interacting with the Meilisearch API (indexing and searching).
4.  **Dockerfile:** Builds a Docker image containing Meilisearch, the Go application (`cmd/mcp`), and necessary dependencies.

## Features

- Hybrid search (vector + keyword) via Meilisearch.
- MCP server for integration with compatible clients.
- Interactive command-line interface for direct searching.
- Optimized Markdown document splitting for embeddings.
- Dockerfile for building application images, potentially including pre-built indexes.

## Components

- **`cmd/mcp`:** The main application binary.
    - `cmd/mcp serve`: Runs the MCP server.
    - `cmd/mcp search`: Runs the interactive search CLI.
    - *(Indexing functionality is handled by utilities, not a dedicated subcommand currently)*
- **`internal/meilisearchutil`:** Utilities for Meilisearch API interactions (indexing, searching, settings).
- **`internal/splitter`:** Document splitting logic, including a Markdown-aware splitter.
- **`Dockerfile`:** Builds the container image.

## Usage

### Recommended Workflow: Pre-built Index Image

The most efficient way to use this project, especially in production or shared environments, is to pre-build your Meilisearch index into a custom Docker image:

1.  **Prepare Data:** Gather the documents you want to index.
2.  **Build Index:** Use the utilities provided in this repository (or build a custom script leveraging `internal/splitter` and `internal/meilisearchutil`) to:
    a. Split documents into chunks.
    b. Generate embeddings for each chunk (using OpenAI or another provider).
    c. Upload the chunks and vectors to a *temporary* Meilisearch instance.
3.  **Create Snapshot:** Create a snapshot of the populated Meilisearch index.
4.  **Build Custom Image:** Modify the provided `Dockerfile` (or create a new one):
    a. Start from the base Meilisearch image or the image built by the provided `Dockerfile`.
    b. Copy the Meilisearch snapshot into the image.
    c. Configure Meilisearch to load from the snapshot on startup.
    d. Include the `cmd/mcp` binary.
5.  **Deploy:** Run your custom image. Meilisearch will start with your data already indexed, and the MCP server or CLI can use it immediately.

This approach avoids costly and time-consuming indexing every time a container starts and ensures data consistency.

### Running the MCP Server

Run the MCP server using a pre-built image (replace `your-custom-built-image:latest` with your image name):

```bash
docker run --rm -p 7777:7777 \ # Expose MCP port
  -e MEILI_HTTP_ADDR="http://<meilisearch_host>:7700" \ # Internal or external Meili address
  -e MEILI_MASTER_KEY="your_meili_master_key" \ 
  -e OPENAI_API_KEY="your_openai_api_key" \ 
  your-custom-built-image:latest serve
```

If Meilisearch is running *within* the same container (as in the default Dockerfile), `MEILI_HTTP_ADDR` might be `http://localhost:7700`.

### Using the Interactive Search CLI

**Option 1: Using Go (Local Development)**

Run against a running Meilisearch instance:

```bash
go run ./cmd/mcp search "your search query" \ 
  --meili-http-addr="http://localhost:7700" \ 
  --meili-api-key="your_meili_master_key" \ 
  --openai-api-key="your_openai_api_key"

# For interactive mode:
go run ./cmd/mcp search -i \ 
  --meili-http-addr="http://localhost:7700" \ 
  --meili-api-key="your_meili_master_key" \ 
  --openai-api-key="your_openai_api_key"
```

**Option 2: Using Docker**

Run the CLI within the container (useful for accessing a pre-built index or running in isolated environments):

```bash
docker run --rm -it \ 
  -e MEILI_HTTP_ADDR="http://<meilisearch_host>:7700" \ 
  -e MEILI_MASTER_KEY="your_meili_master_key" \ 
  -e OPENAI_API_KEY="your_openai_api_key" \ 
  your-custom-built-image:latest search -i 
```

## Configuration (Environment Variables / CLI Flags)

The application (`cmd/mcp`) uses the following configuration options, settable via environment variables (uppercase with underscores) or CLI flags (lowercase with hyphens). CLI flags take precedence.

- **Meilisearch:**
    - `MEILI_HTTP_ADDR` / `--meili-http-addr`: Meilisearch instance address (Default: `http://localhost:7700`)
    - `MEILI_MASTER_KEY` / `--meili-api-key`: Meilisearch master/API key (Default: `masterKey`)
    - `MEILI_INDEX_NAME` / `--meili-index-name`: Target index name (Default: `documents`)
- **Embeddings (OpenAI):**
    - `OPENAI_API_KEY` / `--openai-api-key`: API key for OpenAI embeddings (Required for indexing/searching). (*Other providers may be supported in the future*).
    - `EMBEDDING_MODEL_NAME` / `--embedding-model-name`: Name of the embedding model (Default: `text-embedding-3-small`)
- **Document Splitting:**
    - `SPLIT_CHUNK_SIZE` / `--split-chunk-size`: Max tokens per document chunk (Default: `500`)
    - `SPLIT_CHUNK_OVERLAP` / `--split-chunk-overlap`: Token overlap between chunks (Default: `50`)
- **Application:**
    - `LOG_LEVEL` / `--log-level`: Log level (`debug`, `info`, `warn`, `error`. Default: `info`)
    - `MCP_SERVER_PORT` / `--mcp-server-port`: Port for the MCP server (Default: `7777`)

## Indexing Process

Unlike the previous Python version, indexing is **not** automatic on container startup in the current Go implementation.

You need to index your documents *before* running the MCP server or search CLI for them to be searchable.

The general process involves:

1.  **Reading Documents:** Load your source files (e.g., Markdown from a directory).
2.  **Splitting:** Use the `internal/splitter.MarkdownSplitter` (or a custom splitter) to break documents into manageable chunks based on token limits (`SPLIT_CHUNK_SIZE`) and overlap (`SPLIT_CHUNK_OVERLAP`).
3.  **Embedding:** For each chunk, generate a vector embedding using the configured provider (e.g., OpenAI via `OPENAI_API_KEY` and `EMBEDDING_MODEL_NAME`).
4.  **Uploading:** Use `internal/meilisearchutil` to:
    a. Configure the target Meilisearch index (`MEILI_INDEX_NAME`) with appropriate filterable/sortable attributes and embedding settings.
    b. Upload the document chunks along with their generated vector embeddings to Meilisearch.

Refer to the `internal/meilisearchutil` and `internal/splitter` package code for details on how to implement this. The recommended approach is to perform these steps during a CI/CD pipeline or a dedicated script to build a pre-indexed Docker image (see "Recommended Workflow" above).

## Development

- **Build:** `task build`
- **Test:** `task test`
- **Lint:** `task lint`

## License

MIT License
