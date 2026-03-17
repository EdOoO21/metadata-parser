APP_NAME=catalog

.PHONY: tidy build test run-demo help

help:
	@echo "Доступные команды:"
	@echo "  make tidy      - подтянуть зависимости"
	@echo "  make build     - собрать бинарник"
	@echo "  make test      - запустить тесты"
	@echo "  make run-demo  - проверить bootstrap команды run"

tidy:
	go mod tidy

build:
	go build -o bin/$(APP_NAME) ./cmd/catalog

test:
	go test ./...

run-demo:
	go run ./cmd/catalog run --config ./demo/config/demo.yaml
