APP_NAME=catalog

.PHONY: tidy build test run-demo help

help:
	@echo "Доступные команды:"
	@echo "  make tidy      - подтянуть зависимости"
	@echo "  make build     - собрать бинарник"
	@echo "  make test      - запустить тесты"
	@echo "  make cover     - запустить тесты и создать файл coverage.out и coverage.html для просмотра"
	@echo "  make run-demo  - проверить bootstrap команды run"

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
