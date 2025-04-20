FROM getmeili/meilisearch

# Install dependencies for Python and git
RUN apk update && apk add --no-cache \
    python3 \
    py3-pip \
    git \
    curl \
    jq

# Clone and install the official Python MCP server in a virtual environment
RUN git clone https://github.com/meilisearch/meilisearch-mcp.git /tmp/meilisearch-mcp \
 && python3 -m venv /opt/venv \
 && . /opt/venv/bin/activate \
 && pip install --no-cache-dir /tmp/meilisearch-mcp \
 && rm -rf /tmp/meilisearch-mcp

WORKDIR /app

# Copy entrypoint script
COPY entrypoint.sh ./

# Expose MCP (3000) - Meili (7700) is already exposed by base image
EXPOSE 3000

# Use Tini as PID 1 for proper signal handling & zombie reaping
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["bash", "entrypoint.sh"]