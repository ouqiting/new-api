FRONTEND_DIR = ./web/default
BACKEND_DIR = .
DEV_FRONTEND_DEFAULT_PORT ?= 5173
DEV_COMPOSE_FILE = docker-compose.dev.yml
DEV_POSTGRES_SERVICE = postgres
DEV_BACKEND_SERVICE = new-api
DEV_POSTGRES_DB = new-api
DEV_POSTGRES_USER = root
DEV_SQLITE_PATH ?= one-api.db
BUILD_OUTPUT ?= new-api

.PHONY: all ensure-frontend-assets build build-frontend release start-backend dev dev-api dev-api-rebuild dev-web reset-setup

all: build-frontend start-backend

ensure-frontend-assets:
	@if [ ! -f "$(FRONTEND_DIR)/dist/index.html" ]; then \
		echo "Frontend dist not found; creating placeholder assets for Go embed."; \
		mkdir -p "$(FRONTEND_DIR)/dist"; \
		printf '%s\n' '<!doctype html><html><head><title>Frontend assets not built</title></head><body>Frontend assets are not built. Run make build-frontend before packaging a release.</body></html>' > "$(FRONTEND_DIR)/dist/index.html"; \
	fi

build: ensure-frontend-assets
	@echo "Building backend..."
	@go build -ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=$$(cat VERSION)'" -o $(BUILD_OUTPUT)

build-frontend:
	@echo "Building default frontend..."
	@cd ./web && bun install --frozen-lockfile
	@cd $(FRONTEND_DIR) && DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat ../../VERSION) bun run build

release: build-frontend build

start-backend:
	@echo "Starting backend dev server..."
	@cd $(BACKEND_DIR) && go run main.go &

dev-api:
	@echo "Starting backend services (docker)..."
	@docker compose -f $(DEV_COMPOSE_FILE) up -d

dev-api-rebuild:
	@echo "Rebuilding and starting backend service (docker)..."
	@docker compose -f $(DEV_COMPOSE_FILE) up -d --build $(DEV_BACKEND_SERVICE)

dev-web:
	@echo "Starting frontend dev server..."
	@echo "Default frontend: http://localhost:$(DEV_FRONTEND_DEFAULT_PORT)"
	@cd ./web && bun install
	@cd $(FRONTEND_DIR) && bun run dev -- --host 0.0.0.0 --port $(DEV_FRONTEND_DEFAULT_PORT)

dev: dev-api dev-web

reset-setup:
	@echo "Resetting local setup wizard state..."
	@if docker compose -f $(DEV_COMPOSE_FILE) ps --services --status running | grep -qx "$(DEV_POSTGRES_SERVICE)"; then \
		echo "Detected running docker dev PostgreSQL. Removing setup record and root users..."; \
		docker compose -f $(DEV_COMPOSE_FILE) exec -T $(DEV_POSTGRES_SERVICE) \
			psql -U $(DEV_POSTGRES_USER) -d $(DEV_POSTGRES_DB) \
			-c 'DELETE FROM setups;' \
			-c 'DELETE FROM users WHERE role = 100;' \
			-c "DELETE FROM options WHERE key IN ('SelfUseModeEnabled', 'DemoSiteEnabled');"; \
		echo "Restarting docker dev backend so setup status is recalculated..."; \
		docker compose -f $(DEV_COMPOSE_FILE) restart $(DEV_BACKEND_SERVICE); \
	elif db_path="$${SQLITE_PATH:-$(DEV_SQLITE_PATH)}"; db_path="$${db_path%%\?*}"; [ -f "$$db_path" ]; then \
		db_path="$${SQLITE_PATH:-$(DEV_SQLITE_PATH)}"; \
		db_path="$${db_path%%\?*}"; \
		echo "Detected local SQLite database: $$db_path"; \
		sqlite3 "$$db_path" \
			"DELETE FROM setups; DELETE FROM users WHERE role = 100; DELETE FROM options WHERE key IN ('SelfUseModeEnabled', 'DemoSiteEnabled');"; \
		echo "SQLite setup state reset. Restart the local backend process before testing the setup wizard."; \
	else \
		echo "No running docker dev PostgreSQL or local SQLite database found."; \
		echo "Start the dev stack with 'make dev-api', or set SQLITE_PATH/DEV_SQLITE_PATH to your local SQLite database."; \
		exit 1; \
	fi
