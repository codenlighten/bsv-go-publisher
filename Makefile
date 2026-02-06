.PHONY: help build run stop clean logs test deps

help: ## Show this help message
	@echo "BSV AKUA Broadcast Server - Make Commands"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build Docker images
	docker-compose build

run: ## Start all services
	docker-compose up -d
	@echo "✓ Services started"
	@echo "  API: http://localhost:8080"
	@echo "  MongoDB: localhost:27017"
	@echo ""
	@echo "View logs with: make logs"

run-dev: ## Start with Mongo Express UI
	docker-compose --profile dev up -d
	@echo "✓ Services started (dev mode)"
	@echo "  API: http://localhost:8080"
	@echo "  Mongo Express: http://localhost:8081"
	@echo "  MongoDB: localhost:27017"

stop: ## Stop all services
	docker-compose down

clean: ## Stop and remove volumes
	docker-compose down -v
	@echo "✓ All data removed"

logs: ## View server logs
	docker-compose logs -f bsv-publisher

logs-all: ## View all service logs
	docker-compose logs -f

stats: ## Show UTXO statistics
	@curl -s http://localhost:8080/admin/stats | jq

health: ## Check server health
	@curl -s http://localhost:8080/health | jq

publish: ## Publish test OP_RETURN (usage: make publish DATA=48656c6c6f)
	@curl -X POST http://localhost:8080/publish \
		-H "Content-Type: application/json" \
		-d '{"data":"$(DATA)"}' | jq

status: ## Check broadcast status (usage: make status UUID=xxx)
	@curl -s http://localhost:8080/status/$(UUID) | jq

deps: ## Download Go dependencies
	go mod download

test: ## Run tests
	go test -v ./...

build-local: ## Build local binary
	go build -o bsv-server ./cmd/server

run-local: ## Run locally (requires MongoDB running)
	go run cmd/server/main.go

shell: ## Open shell in running container
	docker exec -it bsv_akua_server sh

mongo-shell: ## Open MongoDB shell
	docker exec -it bsv_akua_db mongosh -u root -p ${MONGO_PASSWORD}

restart: ## Restart the server
	docker-compose restart bsv-publisher

backup-env: ## Backup .env file
	@cp .env .env.backup.$(shell date +%Y%m%d-%H%M%S)
	@echo "✓ Backed up to .env.backup.$(shell date +%Y%m%d-%H%M%S)"
