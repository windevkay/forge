# FLHO Makefile
# Variables
BINARY_NAME=flho
DOCKER_IMAGE=flho
DOCKER_TAG=latest
MAIN_PATH=./cmd/flho
PORT?=4000

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Build flags
LDFLAGS=-ldflags="-w -s"

.PHONY: help build run clean test deps fmt vet docker-build docker-run docker-clean dev

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

# Development targets
build: ## Build the application
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)

run: ## Run the application locally
	$(GOCMD) run $(MAIN_PATH) -PORT=$(PORT)

dev: ## Run the application in development mode with auto-reload (requires air)
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Air not found. Install with: go install github.com/cosmtrek/air@latest"; \
		echo "Falling back to regular run..."; \
		$(MAKE) run; \
	fi

clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

test: ## Run tests
	$(GOTEST) -v ./...

test-coverage: ## Run tests with coverage
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) tidy

fmt: ## Format code
	$(GOFMT) ./...

vet: ## Run go vet
	$(GOCMD) vet ./...

lint: ## Run golangci-lint (requires golangci-lint)
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Docker targets
docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run: ## Run Docker container
	docker run -p $(PORT):$(PORT) --name $(BINARY_NAME) $(DOCKER_IMAGE):$(DOCKER_TAG)

docker-run-detached: ## Run Docker container in detached mode
	docker run -d -p $(PORT):$(PORT) --name $(BINARY_NAME) $(DOCKER_IMAGE):$(DOCKER_TAG)

docker-stop: ## Stop Docker container
	docker stop $(BINARY_NAME) || true

docker-clean: ## Clean Docker artifacts
	docker stop $(BINARY_NAME) || true
	docker rm $(BINARY_NAME) || true
	docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) || true

docker-logs: ## Show Docker container logs
	docker logs -f $(BINARY_NAME)

# Combined targets
all: clean deps fmt vet test build ## Run all checks and build

docker-rebuild: docker-clean docker-build ## Clean and rebuild Docker image

docker-restart: docker-stop docker-run ## Restart Docker container

# Health check
health: ## Check if the application is running
	@curl -f http://localhost:$(PORT)/health || echo "Application not responding"

# Build for different platforms
build-linux: ## Build for Linux
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 $(MAIN_PATH)

build-windows: ## Build for Windows
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)

build-mac: ## Build for macOS
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)

build-all: build-linux build-windows build-mac ## Build for all platforms

# Installation helpers
install-tools: ## Install development tools
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
