.PHONY: all build test lint clean docker-build docker-push help

# Variables
BACKEND_DIR := backend
FRONTEND_DIR := frontend
DOCKER_REGISTRY := docker.io
DOCKER_USERNAME := $(shell echo $$DOCKER_USERNAME)
VERSION := $(shell git describe --tags --always --dirty)
GO_VERSION := 1.21
NODE_VERSION := 20

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

## help: Show this help message
help:
	@echo 'Usage:'
	@echo '  ${YELLOW}make${NC} ${GREEN}<target>${NC}'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} { \
		if (/^[a-zA-Z_-]+:.*?##.*$$/) {printf "  ${YELLOW}%-20s${NC} %s\n", $$1, $$2} \
		else if (/^## .*$$/) {printf "  ${CYAN}%s${NC}\n", substr($$1,4)} \
		}' $(MAKEFILE_LIST)

## Development

## dev: Start development environment with docker-compose
dev:
	@echo "${GREEN}Starting development environment...${NC}"
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml up

## dev-down: Stop development environment
dev-down:
	@echo "${YELLOW}Stopping development environment...${NC}"
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml down

## Backend

## backend-build: Build backend binary
backend-build:
	@echo "${GREEN}Building backend...${NC}"
	cd $(BACKEND_DIR) && go build -ldflags="-s -w -X main.version=$(VERSION)" -o click-lite .

## backend-test: Run backend tests
backend-test:
	@echo "${GREEN}Running backend tests...${NC}"
	cd $(BACKEND_DIR) && go test -v -race -coverprofile=coverage.out ./...

## backend-lint: Lint backend code
backend-lint:
	@echo "${GREEN}Linting backend...${NC}"
	cd $(BACKEND_DIR) && golangci-lint run --timeout=5m

## backend-fmt: Format backend code
backend-fmt:
	@echo "${GREEN}Formatting backend code...${NC}"
	cd $(BACKEND_DIR) && go fmt ./...
	cd $(BACKEND_DIR) && goimports -w .

## backend-run: Run backend locally
backend-run:
	@echo "${GREEN}Running backend...${NC}"
	cd $(BACKEND_DIR) && go run .

## Frontend

## frontend-install: Install frontend dependencies
frontend-install:
	@echo "${GREEN}Installing frontend dependencies...${NC}"
	cd $(FRONTEND_DIR) && pnpm install

## frontend-build: Build frontend
frontend-build:
	@echo "${GREEN}Building frontend...${NC}"
	cd $(FRONTEND_DIR) && pnpm run build

## frontend-test: Run frontend tests
frontend-test:
	@echo "${GREEN}Running frontend tests...${NC}"
	cd $(FRONTEND_DIR) && pnpm test -- --coverage --watchAll=false

## frontend-lint: Lint frontend code
frontend-lint:
	@echo "${GREEN}Linting frontend...${NC}"
	cd $(FRONTEND_DIR) && pnpm run lint

## frontend-fmt: Format frontend code
frontend-fmt:
	@echo "${GREEN}Formatting frontend code...${NC}"
	cd $(FRONTEND_DIR) && pnpm run format

## frontend-run: Run frontend locally
frontend-run:
	@echo "${GREEN}Running frontend...${NC}"
	cd $(FRONTEND_DIR) && pnpm run dev

## Docker

## docker-build: Build all Docker images
docker-build: docker-build-backend docker-build-frontend

## docker-build-backend: Build backend Docker image
docker-build-backend:
	@echo "${GREEN}Building backend Docker image...${NC}"
	docker build -t $(DOCKER_REGISTRY)/$(DOCKER_USERNAME)/click-lite-backend:$(VERSION) \
		-t $(DOCKER_REGISTRY)/$(DOCKER_USERNAME)/click-lite-backend:latest \
		-f $(BACKEND_DIR)/Dockerfile $(BACKEND_DIR)

## docker-build-frontend: Build frontend Docker image
docker-build-frontend:
	@echo "${GREEN}Building frontend Docker image...${NC}"
	docker build -t $(DOCKER_REGISTRY)/$(DOCKER_USERNAME)/click-lite-frontend:$(VERSION) \
		-t $(DOCKER_REGISTRY)/$(DOCKER_USERNAME)/click-lite-frontend:latest \
		-f $(FRONTEND_DIR)/Dockerfile $(FRONTEND_DIR)

## docker-push: Push all Docker images
docker-push: docker-push-backend docker-push-frontend

## docker-push-backend: Push backend Docker image
docker-push-backend:
	@echo "${GREEN}Pushing backend Docker image...${NC}"
	docker push $(DOCKER_REGISTRY)/$(DOCKER_USERNAME)/click-lite-backend:$(VERSION)
	docker push $(DOCKER_REGISTRY)/$(DOCKER_USERNAME)/click-lite-backend:latest

## docker-push-frontend: Push frontend Docker image
docker-push-frontend:
	@echo "${GREEN}Pushing frontend Docker image...${NC}"
	docker push $(DOCKER_REGISTRY)/$(DOCKER_USERNAME)/click-lite-frontend:$(VERSION)
	docker push $(DOCKER_REGISTRY)/$(DOCKER_USERNAME)/click-lite-frontend:latest

## Testing & Quality

## test: Run all tests
test: backend-test frontend-test

## lint: Run all linters
lint: backend-lint frontend-lint

## fmt: Format all code
fmt: backend-fmt frontend-fmt

## security-scan: Run security scans
security-scan:
	@echo "${GREEN}Running security scans...${NC}"
	@which trivy > /dev/null || (echo "${RED}trivy not installed${NC}" && exit 1)
	trivy fs --severity HIGH,CRITICAL .
	cd $(BACKEND_DIR) && gosec ./...
	cd $(FRONTEND_DIR) && pnpm audit --production

## Utilities

## clean: Clean build artifacts
clean:
	@echo "${YELLOW}Cleaning build artifacts...${NC}"
	rm -f $(BACKEND_DIR)/click-lite
	rm -f $(BACKEND_DIR)/coverage.out
	rm -f $(BACKEND_DIR)/coverage.html
	rm -rf $(FRONTEND_DIR)/build
	rm -rf $(FRONTEND_DIR)/coverage

## setup: Setup development environment
setup:
	@echo "${GREEN}Setting up development environment...${NC}"
	@echo "Checking Go version..."
	@go version | grep -q "go$(GO_VERSION)" || echo "${YELLOW}Warning: Go $(GO_VERSION) recommended${NC}"
	@echo "Checking Node version..."
	@node --version | grep -q "v$(NODE_VERSION)" || echo "${YELLOW}Warning: Node $(NODE_VERSION) recommended${NC}"
	@echo "Installing Go tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/cosmtrek/air@latest
	@echo "Installing pnpm..."
	npm install -g pnpm
	@echo "Installing frontend dependencies..."
	cd $(FRONTEND_DIR) && pnpm install
	@echo "${GREEN}Setup complete!${NC}"

## logs: Show logs from docker-compose
logs:
	docker-compose logs -f

## ps: Show running containers
ps:
	docker-compose ps

## init-db: Initialize ClickHouse database
init-db:
	@echo "${GREEN}Initializing ClickHouse database...${NC}"
	docker-compose exec clickhouse clickhouse-client --query "CREATE DATABASE IF NOT EXISTS click_lite"
	docker-compose exec clickhouse clickhouse-client --database click_lite < ./clickhouse/schema.sql

## release: Create a new release
release:
	@echo "${GREEN}Creating release $(VERSION)...${NC}"
	@echo "Building binaries..."
	@make backend-build
	@echo "Building Docker images..."
	@make docker-build
	@echo "Running tests..."
	@make test
	@echo "${GREEN}Release $(VERSION) ready!${NC}"
	@echo "Don't forget to:"
	@echo "  1. git tag -a $(VERSION) -m 'Release $(VERSION)'"
	@echo "  2. git push origin $(VERSION)"
	@echo "  3. make docker-push"

# Default target
all: fmt lint test build