FROM getmeili/meilisearch

# Install dependencies for Python and git
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 \
    python3-pip \
    git \
    curl \
    jq \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Clone and install the official Python MCP server
RUN git clone https://github.com/meilisearch/meilisearch-mcp.git . \
 && pip3 install --no-cache-dir .

# Copy entrypoint script
COPY entrypoint.sh ./

# Expose MCP (3000) - Meili (7700) is already exposed by base image
EXPOSE 3000

# Use Tini as PID 1 for proper signal handling & zombie reaping
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["bash", "entrypoint.sh"]