# Makefile for sniping-bot-v2

.PHONY: help build test clean proto docker-build docker-up docker-down

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
build: ## Build all services
	@echo "Building all services..."
	go build -o bin/grpc-server ./cmd/grpc-server
	go build -o bin/solana-scanner ./cmd/solana-scanner

build-grpc: ## Build gRPC server only
	go build -o bin/grpc-server ./cmd/grpc-server

build-scanner: ## Build Solana scanner only
	go build -o bin/solana-scanner ./cmd/solana-scanner

# Development targets
test: ## Run tests
	go test -v ./...

test-cover: ## Run tests with coverage
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out

# Proto generation
proto: ## Generate protobuf code
	@echo "Generating protobuf code..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/*.proto

# Docker targets
docker-build: ## Build Docker images
	docker build -f docker/grpc-server.Dockerfile -t sniping-bot/grpc-server .
	docker build -f docker/solana-scanner.Dockerfile -t sniping-bot/solana-scanner .

docker-up: ## Start services with docker-compose
	docker-compose up -d

docker-down: ## Stop services with docker-compose
	docker-compose down

# Development helpers
deps: ## Download dependencies
	go mod download
	go mod tidy

fmt: ## Format code
	go fmt ./...

lint: ## Run linter
	golangci-lint run

# Run services locally
run-grpc: ## Run gRPC server locally
	go run ./cmd/grpc-server

run-scanner: ## Run Solana scanner locally
	go run ./cmd/solana-scanner
