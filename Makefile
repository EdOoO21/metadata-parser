APP_NAME=catalog
CATALOG_BIN?=catalog
DIST_DIR?=dist
RELEASE_OS?=$(shell go env GOOS)
RELEASE_ARCH?=$(shell go env GOARCH)
RELEASE_TARGETS?=linux/amd64 linux/arm64 darwin/amd64 darwin/arm64
PG_DEMO_COMPOSE=docker compose -f ./demo/postgres/docker-compose.yml
API_DEMO_COMPOSE=docker compose -f ./demo/api/docker-compose.yml
APP_COMPOSE=docker compose -f ./docker-compose.yml

.PHONY: tidy build test cover run-demo demo-up demo-down-v e2e-demo metadata metadata-diff app-up app-down-v release-cli release-demo release-all help

help:
	@echo "Доступные команды:"
	@echo "  make tidy      - подтянуть зависимости"
	@echo "  make build     - собрать бинарник"
	@echo "  make test      - запустить тесты"
	@echo "  make cover     - запустить тесты и создать файл coverage.out и coverage.html для просмотра"
	@echo "  make app-up    - поднять каталог PostgreSQL и применить миграции"
	@echo "  make app-down-v - остановить каталог PostgreSQL и удалить данные"
	@echo "  make demo-up   - поднять demo PostgreSQL- и API-источники вручную"
	@echo "  make demo-down-v - остановить demo PostgreSQL- и API-источники и удалить данные"
	@echo "  make run-demo  - выполнить run на ./testcases/mixed/1.yaml"
	@echo "  make metadata CATEGORY=files CASE=2 - выполнить run на кейсе, временно подняв только нужные source-контейнеры"
	@echo "  make metadata-diff CATEGORY=files CASE=1 - прогнать baseline/changed и сразу показать diff"
	@echo "  make e2e-demo  - полный demo-сценарий run/report/diff"
	@echo "  make release-cli RELEASE_OS=linux RELEASE_ARCH=amd64 - собрать архив с бинарем"
	@echo "  make release-demo - собрать demo-bundle"
	@echo "  make release-all - собрать demo-bundle и архивы CLI для linux/darwin amd64/arm64"
	@echo "  make demo-down-v - остановить demo-инфраструктуру"
	@echo "  Переменная CATALOG_BIN позволяет указать путь к бинарю, например CATALOG_BIN=./bin/catalog"

tidy:
	go mod tidy

build:
	go build -o bin/$(APP_NAME) ./cmd/catalog

release-cli:
	@mkdir -p "$(DIST_DIR)"
	CGO_ENABLED=0 GOOS=$(RELEASE_OS) GOARCH=$(RELEASE_ARCH) go build -o "$(DIST_DIR)/$(APP_NAME)" ./cmd/catalog
	tar -czf "$(DIST_DIR)/metadata-parser_$(RELEASE_OS)_$(RELEASE_ARCH).tar.gz" -C "$(DIST_DIR)" "$(APP_NAME)"
	rm -f "$(DIST_DIR)/$(APP_NAME)"

release-demo:
	@mkdir -p "$(DIST_DIR)"
	tar -czf "$(DIST_DIR)/metadata-parser-demo.tar.gz" \
		docker-compose.yml \
		demo \
		testcases \
		scripts \
		internal/infrastructure/db/postgres/migrations \
		Makefile \
		README.md \
		.env

release-all:
	@set -e; \
	for target in $(RELEASE_TARGETS); do \
		os="$${target%/*}"; \
		arch="$${target#*/}"; \
		echo "Building $$os/$$arch"; \
		$(MAKE) release-cli RELEASE_OS="$$os" RELEASE_ARCH="$$arch"; \
	done
	@$(MAKE) release-demo

test:
	go test ./...
cover:
	go test -cover -coverprofile=coverage.out ./...

	go tool cover -html=coverage.out -o coverage.html

	explorer.exe coverage.html

run-demo:
	$(MAKE) metadata CATEGORY=mixed CASE=1

metadata:
	@category="$(CATEGORY)"; \
	case_id="$(CASE)"; \
	if [ -z "$$category" ]; then \
		echo "usage: make metadata CATEGORY=<files|postgres|api|mixed|diff_mixed> CASE=<id>"; \
		exit 1; \
	fi; \
	if [ -z "$$case_id" ]; then \
		echo "usage: make metadata CATEGORY=<files|postgres|api|mixed|diff_mixed> CASE=<id>"; \
		exit 1; \
	fi; \
	if [ ! -f "./testcases/$$category/$$case_id.yaml" ]; then \
		echo "config ./testcases/$$category/$$case_id.yaml not found"; \
		exit 1; \
	fi; \
	if [ ! -f ./.env ]; then \
		echo ".env not found. Add it to the project root before running metadata"; \
		exit 1; \
	fi; \
	set -a; \
	. ./.env; \
	set +a; \
	app_bin="$(CATALOG_BIN)"; \
	if ! command -v "$$app_bin" >/dev/null 2>&1; then \
		if [ "$$app_bin" = "catalog" ] && [ -x "./bin/$(APP_NAME)" ]; then \
			app_bin="./bin/$(APP_NAME)"; \
		else \
			echo "catalog binary not found. Install release binary or set CATALOG_BIN=./bin/catalog"; \
			exit 1; \
		fi; \
	fi; \
	wait_catalog_db() { \
		attempts=30; \
		stable_hits=0; \
		while [ $$attempts -gt 0 ]; do \
			if PGPASSWORD="$$CATALOG_DB_PASSWORD" psql "postgres://$$CATALOG_DB_USER:$$CATALOG_DB_PASSWORD@localhost:5432/$$CATALOG_DB_NAME?sslmode=disable" -Atqc "select 1" >/dev/null 2>&1; then \
				stable_hits=$$((stable_hits + 1)); \
				if [ $$stable_hits -ge 3 ]; then \
					return 0; \
				fi; \
			else \
				stable_hits=0; \
			fi; \
			sleep 1; \
			attempts=$$((attempts - 1)); \
		done; \
		echo "catalog_db is not ready for CLI"; \
		return 1; \
	}; \
	wait_pg_case() { \
		db_name="$$1"; \
		attempts=30; \
		while [ $$attempts -gt 0 ]; do \
			if PGPASSWORD="$$CATALOG_DB_PASSWORD" psql "postgres://$$CATALOG_DB_USER:$$CATALOG_DB_PASSWORD@localhost:55433/postgres?sslmode=disable" -Atqc "select 1 from pg_database where datname = '$$db_name'" 2>/dev/null | grep -q 1; then \
				return 0; \
			fi; \
			sleep 1; \
			attempts=$$((attempts - 1)); \
		done; \
		echo "demo postgres source $$db_name is not ready"; \
		return 1; \
	}; \
	wait_api_port() { \
		port="$$1"; \
		attempts=30; \
		while [ $$attempts -gt 0 ]; do \
			if curl -fsS "http://localhost:$$port/openapi.json" >/dev/null 2>&1; then \
				return 0; \
			fi; \
			sleep 1; \
			attempts=$$((attempts - 1)); \
		done; \
		echo "demo api on port $$port is not ready"; \
		return 1; \
	}; \
	if [ "$$category" = "postgres" ]; then \
		trap '$(PG_DEMO_COMPOSE) down -v' EXIT INT TERM; \
		$(PG_DEMO_COMPOSE) up -d source_pg; \
		export DEMO_PG_DSN="postgres://$$CATALOG_DB_USER:$$CATALOG_DB_PASSWORD@localhost:55433/source_case_$$case_id?sslmode=disable"; \
		wait_pg_case "source_case_$$case_id"; \
	elif [ "$$category" = "api" ]; then \
		trap '$(API_DEMO_COMPOSE) down -v' EXIT INT TERM; \
		$(API_DEMO_COMPOSE) up -d demo_api_1 demo_api_2 demo_api_3 demo_api_4 demo_api_5 demo_api_6 demo_api_7 demo_api_8; \
		for port in 8081 8082 8083 8084 8085 8086 8087 8088; do \
			wait_api_port "$$port" || exit 1; \
		done; \
	elif [ "$$category" = "mixed" ] || [ "$$category" = "diff_mixed" ]; then \
		trap '$(PG_DEMO_COMPOSE) down -v; $(API_DEMO_COMPOSE) down -v' EXIT INT TERM; \
		$(PG_DEMO_COMPOSE) up -d source_pg; \
		$(API_DEMO_COMPOSE) up -d demo_api_1 demo_api_2 demo_api_3 demo_api_4 demo_api_5 demo_api_6 demo_api_7 demo_api_8; \
		export DEMO_PG_DSN="postgres://$$CATALOG_DB_USER:$$CATALOG_DB_PASSWORD@localhost:55433/source_case_$$case_id?sslmode=disable"; \
		wait_pg_case "source_case_$$case_id"; \
		for port in 8081 8082 8083 8084 8085 8086 8087 8088; do \
			wait_api_port "$$port" || exit 1; \
		done; \
	fi; \
	wait_catalog_db || exit 1; \
	"$$app_bin" run --config ./testcases/$$category/$$case_id.yaml

metadata-diff:
	@category="$(CATEGORY)"; \
	case_id="$(CASE)"; \
	if [ -z "$$category" ]; then \
		echo "usage: make metadata-diff CATEGORY=<files|postgres|api|mixed> CASE=<id>"; \
		exit 1; \
	fi; \
	if [ -z "$$case_id" ]; then \
		echo "usage: make metadata-diff CATEGORY=<files|postgres|api|mixed> CASE=<id>"; \
		exit 1; \
	fi; \
	if [ ! -f ./.env ]; then \
		echo ".env not found. Add it to the project root before running metadata-diff"; \
		exit 1; \
	fi; \
	CATALOG_BIN="$(CATALOG_BIN)" bash ./scripts/metadata_diff.sh "$$category" "$$case_id"

app-up:
	@if [ ! -f ./.env ]; then echo ".env not found. Add it to the project root before running app-up"; exit 1; fi
	-docker rm -f metadata-parser-catalog_db-1 metadata-parser-catalog_db_migrate-1 >/dev/null 2>&1 || true
	-docker ps -aq --filter name=metadata-parser-catalog_db_migrate-run- | xargs -r docker rm -f >/dev/null 2>&1 || true
	@attempts=10; \
	while [ $$attempts -gt 0 ]; do \
		if ! docker ps -a --format '{{.Names}}' | grep -qx 'metadata-parser-catalog_db-1'; then \
			break; \
		fi; \
		sleep 1; \
		attempts=$$((attempts - 1)); \
	done; \
	if docker ps -a --format '{{.Names}}' | grep -qx 'metadata-parser-catalog_db-1'; then \
		echo "metadata-parser-catalog_db-1 still exists after cleanup"; \
		exit 1; \
	fi
	@attempts=10; \
	while [ $$attempts -gt 0 ]; do \
		if $(APP_COMPOSE) up -d catalog_db; then \
			break; \
		fi; \
		sleep 1; \
		attempts=$$((attempts - 1)); \
	done; \
	if [ $$attempts -eq 0 ]; then \
		echo "catalog_db failed to start"; \
		exit 1; \
	fi
	@set -a; \
	. ./.env; \
	set +a; \
	attempts=30; \
	stable_hits=0; \
	while [ $$attempts -gt 0 ]; do \
		if PGPASSWORD="$$CATALOG_DB_PASSWORD" psql "postgres://$$CATALOG_DB_USER:$$CATALOG_DB_PASSWORD@localhost:5432/$$CATALOG_DB_NAME?sslmode=disable" -Atqc "select 1" >/dev/null 2>&1; then \
			stable_hits=$$((stable_hits + 1)); \
			if [ $$stable_hits -ge 3 ]; then \
				break; \
			fi; \
		else \
			stable_hits=0; \
		fi; \
		sleep 1; \
		attempts=$$((attempts - 1)); \
	done; \
	if [ $$attempts -eq 0 ]; then \
		echo "catalog_db is not ready"; \
		exit 1; \
	fi
	@set -a; \
	. ./.env; \
	set +a; \
	for migration in ./internal/infrastructure/db/postgres/migrations/*.up.sql; do \
		echo "Applying $$migration"; \
		PGPASSWORD="$$CATALOG_DB_PASSWORD" psql "postgres://$$CATALOG_DB_USER:$$CATALOG_DB_PASSWORD@localhost:5432/$$CATALOG_DB_NAME?sslmode=disable" -v ON_ERROR_STOP=1 -f "$$migration" >/dev/null; \
	done
	@set -a; \
	. ./.env; \
	set +a; \
	attempts=30; \
	stable_hits=0; \
	while [ $$attempts -gt 0 ]; do \
		if PGPASSWORD="$$CATALOG_DB_PASSWORD" psql "postgres://$$CATALOG_DB_USER:$$CATALOG_DB_PASSWORD@localhost:5432/$$CATALOG_DB_NAME?sslmode=disable" -Atqc "select 1" >/dev/null 2>&1; then \
			stable_hits=$$((stable_hits + 1)); \
			if [ $$stable_hits -ge 3 ]; then \
				exit 0; \
			fi; \
		else \
			stable_hits=0; \
		fi; \
		sleep 1; \
		attempts=$$((attempts - 1)); \
	done; \
	echo "catalog_db is not ready after migrations"; \
	exit 1

app-down-v:
	$(APP_COMPOSE) down -v --remove-orphans
	-docker rm -f metadata-parser-catalog_db-1 metadata-parser-catalog_db_migrate-1 >/dev/null 2>&1 || true
	-docker ps -aq --filter name=metadata-parser-catalog_db_migrate-run- | xargs -r docker rm -f >/dev/null 2>&1 || true
	@attempts=10; \
	while [ $$attempts -gt 0 ]; do \
		if ! docker ps -a --format '{{.Names}}' | grep -qx 'metadata-parser-catalog_db-1'; then \
			break; \
		fi; \
		sleep 1; \
		attempts=$$((attempts - 1)); \
	done
	-docker volume rm metadata-parser_catalog_db_data >/dev/null 2>&1 || true
	-docker network rm metadata-parser_default >/dev/null 2>&1 || true

demo-up:
	@if [ ! -f ./.env ]; then echo ".env not found. Add it to the project root before running demo-up"; exit 1; fi
	$(PG_DEMO_COMPOSE) up -d source_pg
	$(API_DEMO_COMPOSE) up -d demo_api_1 demo_api_2 demo_api_3 demo_api_4 demo_api_5 demo_api_6 demo_api_7 demo_api_8

demo-down-v:
	$(PG_DEMO_COMPOSE) down -v
	$(API_DEMO_COMPOSE) down -v

e2e-demo:
	@echo "Используй README для пошагового сценария: make app-up, затем metadata/report/diff, затем app-down-v"

%:
	@:
