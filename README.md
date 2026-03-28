# metadata-parser

Учебный CLI-проект каталога метаданных на Go.

## Состав проекта

- CLI с командами `run`, `report`, `diff`
- help и описания команд на русском языке
- загрузка YAML-конфига v1
- валидация обязательных полей конфига
- доменные модели под согласованную схему каталога
- миграция `000001_init` для PostgreSQL-каталога
- PostgreSQL repository для записи и чтения каталога
- файловый CSV-коннектор с профилированием
- Parquet-коннектор с профилированием
- PostgreSQL-коннектор с профилированием
- REST/OpenAPI-коннектор с discovery и профилированием простых `GET` endpoint-ов
- общая фабрика scanner-ов по типу source
- `report` в Markdown и CSV
- `diff` по датасетам и колонкам
- unit-тесты на ключевые слои

В проекте можно прогнать CSV, Parquet, PostgreSQL и REST-источники,
сохранить слепок в каталог, построить отчет по run и сравнить два запуска.

## Требования

- Go 1.23.2+
- Docker и Docker Compose — нужны для локального запуска сервиса и e2e-demo

## Docker Compose

В проекте теперь два compose-контура:
- [`docker-compose.yml`](/home/edo/metadata-parser/docker-compose.yml) — локальный запуск самого приложения и каталога
- [`test/e2e/docker-compose.yml`](/home/edo/metadata-parser/test/e2e/docker-compose.yml) — расширенный demo/e2e стенд с тестовым REST-источником

## Быстрый старт

### 1. Поднять сервис

```bash
docker compose up --build -d
```

После запуска будут доступны:
- `catalog_db` на `localhost:5432`
- контейнер `catalog` с CLI внутри

### 2. Выполнить `run` для файлового источника

```bash
docker compose exec catalog /app/catalog run --config /app/demo/config/files-only.yaml
```

### 3. Построить отчет

```bash
docker compose exec catalog /app/catalog report --config /app/demo/config/files-only.yaml --run-id 1
```

### 4. Выполнить второй `run` и сравнить два запуска

```bash
docker compose exec catalog /app/catalog run --config /app/demo/config/files-only.yaml
docker compose exec catalog /app/catalog diff --config /app/demo/config/files-only.yaml --from-run-id 1 --to-run-id 2
```

### 5. Остановить сервис

```bash
docker compose down -v
```

## Demo e2e

Для сценария с `postgres`, `files` и `rest` demo-источниками:

```bash
docker compose -f ./test/e2e/docker-compose.yml up -d catalog_db demo_api
docker compose -f ./test/e2e/docker-compose.yml --profile migrate up catalog_db_migrate
go run ./cmd/catalog run --config ./demo/config/demo.yaml
go run ./cmd/catalog report --config ./demo/config/demo.yaml --run-id 1
go run ./cmd/catalog run --config ./demo/config/demo.yaml
go run ./cmd/catalog diff --config ./demo/config/demo.yaml --from-run-id 1 --to-run-id 2
docker compose -f ./test/e2e/docker-compose.yml down -v
```

## Тесты

Запуск всех тестов:

```bash
go test ./...
```

Или через Makefile:

```bash
make test
```

## Структура demo-конфига

Файл `demo/config/demo.yaml` описывает:
- каталог PostgreSQL через `CATALOG_DSN`
- исходный PostgreSQL-источник `source_demo` через `DEMO_PG_DSN`
- файловый источник через путь `./demo/files` с глубиной обхода `max_depth`
- REST API через `base_url` и OpenAPI discovery

Файл [`test/e2e/docker-compose.yml`](/home/edo/metadata-parser/test/e2e/docker-compose.yml) нужен только для локального demo/e2e сценария и не является частью продового окружения.

Для простого локального запуска только файлового источника можно использовать корневой compose и конфиг [`demo/config/files-only.yaml`](/home/edo/metadata-parser/demo/config/files-only.yaml).
