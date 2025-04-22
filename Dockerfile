# Builder stage
FROM golang:1.23.4-alpine AS builder
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o meilisearch-mcp ./cmd/meilisearch-mcp

# Final stage
FROM getmeili/meilisearch:v1.14.0

# Install dependencies
RUN apk update && \
    # Add community repository for uv package
    echo "http://dl-cdn.alpinelinux.org/alpine/v3.21/community" >> /etc/apk/repositories && \
    apk update && \
    apk add --no-cache \
    bash \
    git \
    curl \
    jq \
    python3 \
    py3-pip \
    uv

# Clone and install the official Python MCP server using uv
RUN git clone https://github.com/meilisearch/meilisearch-mcp.git /app/meilisearch-mcp \
 && cd /app/meilisearch-mcp \
 && git checkout 5fce6a7a94b9c2e8a6007f4d80e5162c0a9eccb5 # Latest commit hash of main as of 2025-04-20 \

WORKDIR /app

# Set default master key (not sensitive data)
ENV MEILI_MASTER_KEY=masterKey

# Copy entrypoint script
COPY entrypoint.sh ./

# Copy meilisearch-mcp from builder stage
COPY --from=builder /workspace/meilisearch-mcp /usr/local/bin/meilisearch-mcp

# Use Tini as PID 1 for proper signal handling & zombie reaping
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["meilisearch-mcp", "serve"]