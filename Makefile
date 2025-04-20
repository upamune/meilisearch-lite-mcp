# Makefile for Meilisearch + MCP Server

# Variables
IMAGE_NAME = meilisearch-lite-mcp
IMAGE_TAG = latest
# Local image name (for local builds)
LOCAL_IMAGE_NAME = $(IMAGE_NAME):$(IMAGE_TAG)
# Remote image name (for pulling from registry)
REMOTE_IMAGE_NAME = ghcr.io/upamune/$(IMAGE_NAME):$(IMAGE_TAG)
# Default to using local image
FULL_IMAGE_NAME = $(LOCAL_IMAGE_NAME)
CONTAINER_NAME = $(IMAGE_NAME)

# Ports
MEILI_PORT = 8777
MCP_PORT = 8300

# Environment variables
MEILI_MASTER_KEY = masterKey
DOCUMENT_DIRS = /app/example/spec,/app/example/guide
CHECK_RETRIES = 30

# Default target
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make build      - Build Docker image locally"
	@echo "  make pull       - Pull Docker image from remote registry"
	@echo "  make run        - Run Docker container using local image"
	@echo "  make run-remote - Run Docker container using remote image"
	@echo "  make run-it     - Run container interactively (for debugging)"
	@echo "  make stop       - Stop running container"
	@echo "  make logs       - View container logs"
	@echo "  make status     - Check container status"
	@echo "  make clean      - Remove Docker image"
	@echo "  make all        - Build image locally and run container"
	@echo "  make restart    - Restart the container"
	@echo "  make help       - Show this help message"

# Build Docker image locally
.PHONY: build
build:
	@echo "Building Docker image locally: $(LOCAL_IMAGE_NAME)"
	docker build -t $(LOCAL_IMAGE_NAME) .

# Pull Docker image from remote registry
.PHONY: pull
pull:
	@echo "Pulling Docker image from remote registry: $(REMOTE_IMAGE_NAME)"
	docker pull $(REMOTE_IMAGE_NAME)

# Run Docker container using local image
.PHONY: run
run:
	@echo "Running Docker container using local image: $(LOCAL_IMAGE_NAME)"
	@if docker ps -a | grep -q $(CONTAINER_NAME); then \
		echo "Container $(CONTAINER_NAME) is already running or exists. Use 'make restart' instead."; \
	else \
		docker run --rm \
			--name $(CONTAINER_NAME) \
			-p $(MEILI_PORT):7700 \
			-p $(MCP_PORT):3000 \
			-e MEILI_MASTER_KEY=$(MEILI_MASTER_KEY) \
			-e DOCUMENT_DIRS="$(DOCUMENT_DIRS)" \
			-e CHECK_RETRIES=$(CHECK_RETRIES) \
			-v $(PWD)/example/spec:/app/example/spec \
			-v $(PWD)/example/guide:/app/example/guide \
			$(LOCAL_IMAGE_NAME); \
	fi
	@echo "Note: When container starts, Meilisearch will be available at: http://localhost:$(MEILI_PORT)"
	@echo "Note: When container starts, MCP server will be available at: http://localhost:$(MCP_PORT)"

# Run Docker container using remote image
.PHONY: run-remote
run-remote:
	@echo "Running Docker container using remote image: $(REMOTE_IMAGE_NAME)"
	@if docker ps -a | grep -q $(CONTAINER_NAME); then \
		echo "Container $(CONTAINER_NAME) is already running or exists. Use 'make restart' instead."; \
	else \
		docker run --pull always --rm \
			--name $(CONTAINER_NAME) \
			-p $(MEILI_PORT):7700 \
			-p $(MCP_PORT):3000 \
			-e MEILI_MASTER_KEY=$(MEILI_MASTER_KEY) \
			-e DOCUMENT_DIRS="$(DOCUMENT_DIRS)" \
			-e CHECK_RETRIES=$(CHECK_RETRIES) \
			-v $(PWD)/example/spec:/app/example/spec \
			-v $(PWD)/example/guide:/app/example/guide \
			$(REMOTE_IMAGE_NAME); \
	fi
	@echo "Note: When container starts, Meilisearch will be available at: http://localhost:$(MEILI_PORT)"
	@echo "Note: When container starts, MCP server will be available at: http://localhost:$(MCP_PORT)"

# Run Docker container interactively (for debugging)
.PHONY: run-it
run-it:
	@echo "Running Docker container interactively: $(LOCAL_IMAGE_NAME)"
	@if docker ps -a | grep -q $(CONTAINER_NAME); then \
		echo "Container $(CONTAINER_NAME) is already running or exists. Use 'make restart' instead."; \
	else \
		docker run -it \
			--name $(CONTAINER_NAME) \
			-p $(MEILI_PORT):7700 \
			-p $(MCP_PORT):3000 \
			-e MEILI_MASTER_KEY=$(MEILI_MASTER_KEY) \
			-e DOCUMENT_DIRS="$(DOCUMENT_DIRS)" \
			-e CHECK_RETRIES=$(CHECK_RETRIES) \
			-v $(PWD)/example/spec:/app/example/spec \
			-v $(PWD)/example/guide:/app/example/guide \
			$(LOCAL_IMAGE_NAME) \
			/bin/bash; \
	fi

# Stop running container
.PHONY: stop
stop:
	@echo "Stopping Docker container with image: $(FULL_IMAGE_NAME)"
	@CONTAINER_ID=$$(docker ps -q --filter ancestor=$(FULL_IMAGE_NAME) | head -n 1); \
	if [ -n "$$CONTAINER_ID" ]; then \
		docker stop $$CONTAINER_ID; \
	else \
		echo "No running container found for image $(FULL_IMAGE_NAME)"; \
	fi

# View container logs
.PHONY: logs
logs:
	@echo "Viewing logs for container with image: $(FULL_IMAGE_NAME)"
	@CONTAINER_ID=$$(docker ps -q --filter ancestor=$(FULL_IMAGE_NAME) | head -n 1); \
	if [ -n "$$CONTAINER_ID" ]; then \
		docker logs $$CONTAINER_ID; \
	else \
		echo "No running container found for image $(FULL_IMAGE_NAME)"; \
	fi

# Check container status
.PHONY: status
status:
	@echo "Checking status of all containers:"
	docker ps

# Restart container
.PHONY: restart
restart: stop run

# Remove Docker image
.PHONY: clean
clean: stop
	@echo "Removing Docker image: $(FULL_IMAGE_NAME)"
	docker rmi $(FULL_IMAGE_NAME) 2>/dev/null || echo "Image not found"

# Build and run in one command
.PHONY: all
all: build run
