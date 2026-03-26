# catalog-tool

Учебный CLI-проект каталога метаданных на Go.

## Что уже работает

- CLI с командами `run`, `report`, `diff`
- help и описания команд на русском языке
- загрузка YAML-конфига v1
- валидация обязательных полей конфига
- доменные модели под согласованную схему каталога
- миграция `000001_init` для PostgreSQL-каталога
- PostgreSQL repository для записи и чтения каталога
- файловый CSV-коннектор с профилированием
- базовый `report` в Markdown и CSV
- unit-тесты на ключевые слои

## Что пока не реализовано

- Parquet-коннектор
- обход PostgreSQL-источников
- discovery и профилирование REST/OpenAPI
- полноценный `diff`

Сейчас это **MVP-версия**, на которой уже можно прогнать CSV-источник,
сохранить слепок в каталог и построить базовый отчет по run.

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

### 4. Выполнить `run`

```bash
go run ./cmd/catalog run --config ./demo/config/demo.yaml
```

Ожидаемый результат:
- конфиг успешно читается;
- CSV-источники профилируются;
- слепок записывается в каталог;
- в stdout печатается `run_id`.

### 5. Построить отчет по run

```bash
go run ./cmd/catalog report --config ./demo/config/demo.yaml --run-id 1
go run ./cmd/catalog report --config ./demo/config/demo.yaml --run-id 1 --output ./report.md --csv-output ./report.csv
```

Ожидаемый результат:
- в stdout или файл строится Markdown-карта источников, датасетов и колонок;
- при `--csv-output` создается плоский CSV-экспорт колонок.

### 6. Сравнить два запуска

```bash
go run ./cmd/catalog diff --config ./demo/config/demo.yaml --from-run-id 1 --to-run-id 2
```

Ожидаемый результат:
- в stdout выводятся новые и удаленные датасеты;
- показываются новые и удаленные колонки;
- отдельно выводятся изменения типа, nullable и комментария.

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

Следующим шагом будут:
- Parquet-коннектор
- PostgreSQL-коннектор
- REST-коннектор
- развитие `diff` до сравнения статистики и top values
