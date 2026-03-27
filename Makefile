APP_NAME=catalog
E2E_COMPOSE=docker compose -f ./test/e2e/docker-compose.yml
APP_COMPOSE=docker compose -f ./docker-compose.yml

.PHONY: tidy build test run-demo demo-up demo-migrate demo-down e2e-demo help

help:
	@echo "Доступные команды:"
	@echo "  make tidy      - подтянуть зависимости"
	@echo "  make build     - собрать бинарник"
	@echo "  make test      - запустить тесты"
	@echo "  make cover     - запустить тесты и создать файл coverage.out и coverage.html для просмотра"
	@echo "  make app-up    - поднять каталог-базу и app compose"
	@echo "  make app-migrate - применить миграции каталога в app compose"
	@echo "  make demo-up   - поднять demo-инфраструктуру"
	@echo "  make demo-migrate - применить миграции каталога"
	@echo "  make run-demo  - выполнить run на demo-конфиге"
	@echo "  make e2e-demo  - полный demo-сценарий run/report/diff"
	@echo "  make demo-down - остановить demo-инфраструктуру"

tidy:
	go mod tidy

build:
	go build -o bin/$(APP_NAME) ./cmd/catalog

test:
	go test ./...
cover:
	go test -cover -coverprofile=coverage.out ./...

	go tool cover -html=coverage.out -o coverage.html

	explorer.exe coverage.html

run-demo:
	go run ./cmd/catalog run --config ./demo/config/demo.yaml

app-up:
	$(APP_COMPOSE) up -d catalog_db

app-migrate:
	$(APP_COMPOSE) --profile migrate up catalog_db_migrate

demo-up:
	$(E2E_COMPOSE) up -d catalog_db demo_api

demo-migrate:
	$(E2E_COMPOSE) --profile migrate up catalog_db_migrate

demo-down:
	$(E2E_COMPOSE) down -v

e2e-demo:
	@echo "Используй README для пошагового сценария или прогоняй команды вручную после demo-up и demo-migrate"
