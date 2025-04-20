FROM python:3.10-alpine

# Install runtime dependencies (including libgcc for Meilisearch) and Tini for PID 1 duties
RUN apk add --no-cache \
      bash \
      curl \
      git \
      libgcc \
      tini

WORKDIR /app

# Clone and install the official Python MCP server (requires Python ≥3.10)
RUN git clone https://github.com/meilisearch/meilisearch-mcp.git . \
 && pip install --no-cache-dir .

# Download Meilisearch binary from GitHub (v1.14.0 example)
ENV MEILI_VERSION=1.14.0
RUN curl -L \
      https://github.com/meilisearch/meilisearch/releases/download/v${MEILI_VERSION}/meilisearch-linux-amd64 \
      -o /usr/local/bin/meilisearch \
 && chmod +x /usr/local/bin/meilisearch

# Copy entrypoint script
COPY entrypoint.sh ./

# Expose Meili (7700) and MCP (3000)
EXPOSE 7700 3000

# Use Tini as PID 1 for proper signal handling & zombie reaping
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["bash", "entrypoint.sh"]