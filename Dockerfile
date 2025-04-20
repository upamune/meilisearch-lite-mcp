FROM getmeili/meilisearch:v1.14.0

# Install dependencies for Python and git
RUN apk update && apk add --no-cache \
    python3 \
    py3-pip \
    git \
    curl \
    jq

# Clone and install the official Python MCP server in a virtual environment
RUN git clone https://github.com/meilisearch/meilisearch-mcp.git /tmp/meilisearch-mcp \
 && cd /tmp/meilisearch-mcp \
 && git checkout 5fce6a7a94b9c2e8a6007f4d80e5162c0a9eccb5 # Latest commit hash of main as of 2025-04-20 \
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