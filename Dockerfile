# Builder stage
FROM golang:1.24 AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /usr/local/bin/mcp ./cmd/mcp

# Final stage
FROM getmeili/meilisearch:latest
COPY --from=builder /usr/local/bin/mcp /usr/local/bin/mcp
COPY scripts/wait-for-meili.sh /usr/local/bin/
ENV MEILI_MASTER_KEY=""
ENTRYPOINT ["meilisearch"]