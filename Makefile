APP_NAME=catalog

.PHONY: tidy build run-demo

tidy:
	go mod tidy

build:
	go build -o bin/$(APP_NAME) ./cmd/catalog

run-demo:
	go run ./cmd/catalog run --config ./demo/config/demo.yaml
