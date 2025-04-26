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
    apk add --no-cache \
    bash \
    git \
    curl \
    jq

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