.PHONY: help up down restart logs clean build test test-api test-graceful-shutdown db-init db-reset ps health

# Default target
.DEFAULT_GOAL := help

# Docker Compose file location
COMPOSE_FILE := docker-compose.yaml
COMPOSE := docker compose -f $(COMPOSE_FILE)

# Colors for output
GREEN  := \033[0;32m
YELLOW := \033[0;33m
RED    := \033[0;31m
BLUE   := \033[0;34m
NC     := \033[0m # No Color

##@ General

help: ## Display this help message
	@echo "$(BLUE)ThinkPixel Local Development Environment$(NC)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make $(YELLOW)<target>$(NC)\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  $(YELLOW)%-25s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n$(BLUE)%s$(NC)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Docker Compose

up: ## Start all services
	@echo "$(GREEN)Starting all services...$(NC)"
	$(COMPOSE) up -d
	@echo "$(GREEN)Services started! Waiting for health checks...$(NC)"
	@sleep 5
	@$(MAKE) health

up-build: ## Build and start all services
	@echo "$(GREEN)Building and starting all services...$(NC)"
	$(COMPOSE) up -d --build
	@echo "$(GREEN)Services started! Waiting for health checks...$(NC)"
	@sleep 5
	@$(MAKE) health

down: ## Stop all services
	@echo "$(YELLOW)Stopping all services...$(NC)"
	$(COMPOSE) down

down-volumes: ## Stop all services and remove volumes (WARNING: data loss!)
	@echo "$(RED)Stopping all services and removing volumes...$(NC)"
	@read -p "This will DELETE ALL DATA. Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		$(COMPOSE) down -v; \
		echo "$(RED)Volumes removed!$(NC)"; \
	else \
		echo "$(GREEN)Aborted.$(NC)"; \
	fi

restart: down up ## Restart all services

restart-api: ## Restart only API Gateway
	@echo "$(YELLOW)Restarting API Gateway...$(NC)"
	$(COMPOSE) restart api-gateway
	@sleep 2
	@$(MAKE) health-api

ps: ## Show running services
	@$(COMPOSE) ps

logs: ## Tail logs from all services
	$(COMPOSE) logs -f

logs-api: ## Tail logs from API Gateway
	$(COMPOSE) logs -f api-gateway

logs-mysql: ## Tail logs from MySQL
	$(COMPOSE) logs -f mysql

logs-nats: ## Tail logs from NATS
	$(COMPOSE) logs -f nats

logs-model: ## Tail logs from ML Model
	$(COMPOSE) logs -f mpnetv2

##@ Building

build: ## Build all services
	@echo "$(GREEN)Building all services...$(NC)"
	$(COMPOSE) build

build-api: ## Build only API Gateway
	@echo "$(GREEN)Building API Gateway...$(NC)"
	$(COMPOSE) build api-gateway

build-no-cache: ## Build all services without cache
	@echo "$(GREEN)Building all services (no cache)...$(NC)"
	$(COMPOSE) build --no-cache

##@ Database

db-init: ## Initialize database with schema
	@echo "$(GREEN)Initializing database...$(NC)"
	@sleep 2
	@docker exec -i mysql mysql -uroot -proot thinkpixel < ../database/db_thinkpixel.sql
	@echo "$(GREEN)Applying patches (if schema version requires them)...$(NC)"
	@# Patches are only needed for upgrading existing databases, skip for fresh installs
	@echo "$(GREEN)Inserting local development data...$(NC)"
	@docker exec -i mysql mysql -uroot -proot thinkpixel < init-db.sql
	@echo "$(GREEN)Database initialized!$(NC)"

db-shell: ## Open MySQL shell
	@docker exec -it mysql mysql -uroot -proot thinkpixel

db-reset: ## Reset database (WARNING: data loss!)
	@echo "$(RED)Resetting database...$(NC)"
	@read -p "This will DELETE ALL DATA. Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker exec -i mysql mysql -uroot -proot -e "DROP DATABASE IF EXISTS thinkpixel; CREATE DATABASE thinkpixel;"; \
		$(MAKE) db-init; \
		echo "$(GREEN)Database reset complete!$(NC)"; \
	else \
		echo "$(GREEN)Aborted.$(NC)"; \
	fi

db-backup: ## Backup database to local/backups/
	@mkdir -p backups
	@echo "$(GREEN)Backing up database...$(NC)"
	@docker exec mysql mysqldump -uroot -proot thinkpixel > backups/thinkpixel_$$(date +%Y%m%d_%H%M%S).sql
	@echo "$(GREEN)Backup saved to backups/$(NC)"

##@ Testing

test: ## Run all tests
	@echo "$(GREEN)Running all tests...$(NC)"
	@$(MAKE) test-health
	@$(MAKE) test-api
	@echo "$(GREEN)All tests passed!$(NC)"

test-health: ## Test service health
	@echo "$(GREEN)Testing service health...$(NC)"
	@$(MAKE) health

test-api: ## Test API endpoints
	@echo "$(GREEN)Testing API Gateway endpoints...$(NC)"
	@bash tests/test-api.sh

test-store: ## Test Store handler with soft-fail behavior
	@echo "$(GREEN)Testing Store handler...$(NC)"
	@bash tests/test-store-handler.sh

test-db-nats: ## Test database connection pooling and NATS error handling
	@echo "$(GREEN)Testing database & NATS error handling...$(NC)"
	@bash tests/test-database-nats.sh

test-metrics: ## Test Prometheus metrics endpoint
	@echo "$(GREEN)Testing Prometheus metrics...$(NC)"
	@bash tests/test-metrics.sh

test-graceful-shutdown: ## Test graceful shutdown
	@echo "$(GREEN)Testing graceful shutdown...$(NC)"
	@cd tests && bash test-graceful-shutdown.sh

test-integration: ## Run integration tests
	@echo "$(GREEN)Running integration tests...$(NC)"
	@bash tests/test-integration.sh

test-unit: ## Run all unit tests
	@echo "$(GREEN)Running unit tests...$(NC)"
	@cd ../docker/api_gateway/api_gateway && go test -v ./document_queue/ ./auth/ ./utils/ ./config/ ./logger/

test-unit-coverage: ## Run unit tests with coverage report
	@echo "$(GREEN)Running unit tests with coverage...$(NC)"
	@cd ../docker/api_gateway/api_gateway && go test -cover ./document_queue/ ./auth/ ./utils/ ./config/ ./logger/

test-unit-all: ## Run all unit tests (all packages)
	@echo "$(GREEN)Running all unit tests...$(NC)"
	@cd ../docker/api_gateway/api_gateway && go test -v ./...

test-unit-coverage-all: ## Run all unit tests with coverage report
	@echo "$(GREEN)Running all unit tests with coverage...$(NC)"
	@cd ../docker/api_gateway/api_gateway && go test -cover ./...

test-unit-verbose: ## Run unit tests with verbose output and coverage
	@echo "$(GREEN)Running unit tests (verbose + coverage)...$(NC)"
	@cd ../docker/api_gateway/api_gateway && go test -v -cover ./document_queue/ ./auth/ ./utils/ ./config/ ./logger/

test-unit-coverage-html: ## Generate HTML coverage report
	@echo "$(GREEN)Generating HTML coverage report...$(NC)"
	@cd ../docker/api_gateway/api_gateway && \
		go test -coverprofile=coverage.out ./... && \
		go tool cover -html=coverage.out -o coverage.html && \
		echo "$(GREEN)Coverage report generated: coverage.html$(NC)" && \
		echo "$(BLUE)Open with: xdg-open ../docker/api_gateway/api_gateway/coverage.html$(NC)"

##@ Health Checks

health: health-mysql health-nats health-qdrant health-model health-api ## Check health of all services

health-mysql: ## Check MySQL health
	@printf "MySQL:        "
	@if docker exec mysql mysqladmin ping -h localhost --silent 2>/dev/null; then \
		echo "$(GREEN)✓ healthy$(NC)"; \
	else \
		echo "$(RED)✗ unhealthy$(NC)"; \
	fi

health-nats: ## Check NATS health
	@printf "NATS:         "
	@if curl -sf http://localhost:8222/varz > /dev/null 2>&1; then \
		echo "$(GREEN)✓ healthy$(NC)"; \
	else \
		echo "$(RED)✗ unhealthy$(NC)"; \
	fi

health-qdrant: ## Check Qdrant health
	@printf "Qdrant:       "
	@if curl -sf http://localhost:6333/healthz > /dev/null 2>&1; then \
		echo "$(GREEN)✓ healthy$(NC)"; \
	else \
		echo "$(RED)✗ unhealthy$(NC)"; \
	fi

health-model: ## Check ML Model health
	@printf "ML Model:     "
	@if curl -sf http://localhost:8000/ping > /dev/null 2>&1; then \
		echo "$(GREEN)✓ healthy$(NC)"; \
	else \
		echo "$(RED)✗ unhealthy$(NC)"; \
	fi

health-api: ## Check API Gateway health
	@printf "API Gateway:  "
	@if curl -sf http://localhost:8080/ping > /dev/null 2>&1; then \
		echo "$(GREEN)✓ healthy$(NC)"; \
	else \
		echo "$(RED)✗ unhealthy$(NC)"; \
	fi

health-redis: ## Check Redis health
	@printf "Redis:        "
	@if docker exec redis-primary redis-cli ping > /dev/null 2>&1; then \
		echo "$(GREEN)✓ healthy$(NC)"; \
	else \
		echo "$(RED)✗ unhealthy$(NC)"; \
	fi

##@ Debugging

shell-api: ## Open shell in API Gateway container
	@docker exec -it api-gateway /bin/sh

shell-mysql: ## Open shell in MySQL container
	@docker exec -it mysql /bin/bash

shell-nats: ## Open shell in NATS container (limited - scratch image)
	@echo "$(YELLOW)NATS uses a scratch image. Use 'make logs-nats' instead.$(NC)"

redis-cli: ## Open Redis CLI (connects to primary via Sentinel)
	@echo "$(BLUE)Connecting to Redis primary via Sentinel...$(NC)"
	@docker exec -it redis-primary redis-cli

redis-sentinel-cli: ## Open Redis Sentinel CLI
	@echo "$(BLUE)Connecting to Redis Sentinel...$(NC)"
	@docker exec -it redis-sentinel redis-cli -p 26379

redis-info: ## Show Redis Sentinel info
	@echo "$(BLUE)Redis Sentinel Status:$(NC)"
	@docker exec redis-sentinel redis-cli -p 26379 SENTINEL masters
	@echo ""
	@echo "$(BLUE)Redis Primary Info:$(NC)"
	@docker exec redis-primary redis-cli INFO replication

qdrant-collections: ## List Qdrant collections
	@curl -s http://localhost:6333/collections | jq

nats-status: ## Show NATS status
	@curl -s http://localhost:8222/varz | jq '{server_name, version, connections, in_msgs, out_msgs, in_bytes, out_bytes}'

##@ Development

dev: up db-init health ## Start development environment (full setup)
	@echo ""
	@echo "$(GREEN)Development environment ready!$(NC)"
	@echo ""
	@echo "$(BLUE)Available services:$(NC)"
	@echo "  API Gateway:     http://localhost:8080"
	@echo "  Mock Site:       http://localhost:8081"
	@echo "  MySQL:           localhost:3306"
	@echo "  Redis Sentinel:  localhost:26380"
	@echo "  Qdrant:          http://localhost:6333"
	@echo "  NATS:            nats://localhost:4222"
	@echo "  NATS Monitor:    http://localhost:8222"
	@echo "  ML Model:        http://localhost:8000"
	@echo ""
	@echo "$(BLUE)Common commands:$(NC)"
	@echo "  make logs-api     - View API Gateway logs"
	@echo "  make test-api     - Run API tests"
	@echo "  make db-shell     - Open MySQL shell"
	@echo "  make redis-cli    - Open Redis CLI"
	@echo "  make clean        - Stop and clean everything"
	@echo ""

watch-logs: ## Watch all logs with color highlighting
	$(COMPOSE) logs -f --tail=50 | grep --color=auto -E "ERROR|WARNING|INFO|$$"

reload-api: build-api restart-api ## Rebuild and restart API Gateway

##@ Cleanup

clean: down ## Stop services and clean up
	@echo "$(YELLOW)Cleaning up...$(NC)"
	@docker system prune -f
	@echo "$(GREEN)Cleanup complete!$(NC)"

clean-all: down-volumes ## Stop services, remove volumes, and clean up everything
	@echo "$(RED)Cleaning everything...$(NC)"
	@docker system prune -af
	@echo "$(GREEN)Complete cleanup done!$(NC)"

##@ Monitoring

stats: ## Show container stats
	@docker stats --no-stream api-gateway mysql nats qdrant mpnetv2

monitor: ## Continuous monitoring of containers
	@docker stats api-gateway mysql nats qdrant mpnetv2

##@ Quick Actions

quick-test: up health test-api ## Quick test: start, health check, and test API

reset: down-volumes dev ## Complete reset: remove everything and start fresh
