# catalog-tool

Стартовый scaffold проекта каталога метаданных на Go.

## Что уже работает

- CLI с командами `run`, `report`, `diff`
- help и описания команд на русском языке
- загрузка YAML-конфига v1
- валидация обязательных полей конфига
- доменные модели под согласованную схему каталога
- миграция `000001_init` для PostgreSQL-каталога
- заготовка PostgreSQL pool/repository
- базовые unit-тесты для конфигурации

## Что пока не реализовано

- реальная запись слепка в PostgreSQL
- обход файловых источников и профилирование CSV/Parquet
- обход PostgreSQL-источников
- discovery и профилирование REST/OpenAPI
- полноценные `report` и `diff`

Сейчас это **bootstrap-версия**, на которой уже можно проверить каркас проекта,
CLI и формат конфига.

## Требования

- Go 1.23.2+
- Docker и Docker Compose — понадобятся на следующем этапе для PostgreSQL-каталога

## Быстрый старт

### 1. Подтянуть зависимости

```bash
go mod tidy
```

### 2. Собрать проект

```bash
go build -o bin/catalog ./cmd/catalog
```

### 3. Посмотреть help

```bash
./bin/catalog --help
./bin/catalog run --help
./bin/catalog report --help
./bin/catalog diff --help
```

### 4. Проверить bootstrap `run`

```bash
go run ./cmd/catalog run --config ./demo/config/demo.yaml
```

Ожидаемый результат:
- конфиг успешно читается;
- в лог выводятся найденные источники;
- в stdout печатается строка про успешное завершение bootstrap-этапа.

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
- исходный PostgreSQL-источник через `DEMO_PG_DSN`
- файловый источник через путь `./demo/files` с глубиной обхода `max_depth`
- REST API через `base_url` и OpenAPI discovery

## Следующий этап

Следующим шагом будет реализован первый рабочий vertical slice:
- PostgreSQL-каталог
- миграции
- `run`
- files source
- CSV parsing
- profiling
- запись snapshot в БД
