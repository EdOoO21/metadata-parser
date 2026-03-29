# metadata-parser

CLI-проект каталога метаданных на Go.

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

- Docker и Docker Compose

## Быстрый старт

Основной сценарий работы такой:
- установить CLI
- скачать проект в виде demo-bundle
- при необходимости поправить подключения в `.env`
- поднять каталоговую БД командой `make app-up`
- запускать кейсы через `catalog ...` или `make metadata ...`
- в конце остановить каталог командой `make app-down-v`

### 1. Установить CLI и скачать demo-bundle

```bash
curl -fsSL https://raw.githubusercontent.com/EdOoO21/metadata-parser/main/scripts/install_release.sh | bash
curl -fsSL https://raw.githubusercontent.com/EdOoO21/metadata-parser/main/scripts/install_demo_bundle.sh | bash
source ~/.bashrc
cd ~/metadata-parser-demo
```

`catalog` - алиас для утилиты парсера метаданных

Если `catalog` после установки не находится, открой новый терминал
или выполни:

```bash
source ~/.bashrc
```

Если проект уже лежит локально, можно пропустить скачивание demo-bundle и просто перейти в его корень.
Если `catalog` уже установлен, можно пропустить и установку CLI:

```bash
cd metadata-parser
```

### 2. Проверить `.env`

В `.env` должны быть заполнены эти ключи:
- `CATALOG_DSN`
- `CATALOG_DB_NAME`
- `CATALOG_DB_USER`
- `CATALOG_DB_PASSWORD`
- `DEMO_PG_DSN`

Менять нужно только значения этих ключей внутри `.env`, не их названия.
Подключения настраиваются только через `.env`.
`Makefile`, скрипты и сам `catalog` автоматически подхватывают этот файл,
если он лежит в текущей рабочей директории.

### 3. Поднять каталог PostgreSQL

```bash
make app-up
```

После запуска будут доступны:
- `catalog_db` на `localhost:5432`
- примененные миграции каталога

### 4. Запустить кейс

Бинарь `catalog` после 1 пункта уже установлен, можно запускать команды напрямую:

```bash
catalog run --config ./testcases/files/1.yaml
```

Для demo-кейсов удобнее использовать `make metadata`:

```bash
make metadata CATEGORY=postgres CASE=1
make metadata CATEGORY=api CASE=1
make metadata CATEGORY=mixed CASE=1
```

Как это работает:
- `catalog run --config ...` можно запускать напрямую, если source уже доступен
- `make metadata CATEGORY=... CASE=...` удобно для demo-кейсов
- для `postgres`, `api`, `mixed`, `diff_mixed` команда сама поднимает нужные source-контейнеры
- после завершения она удаляет только demo source-контейнеры и их данные
- каталоговая БД приложения при этом остается запущенной

Данные, которые уже сохранились в каталоговой БД на `localhost:5432`, можно после этого проверять вручную через `report` или `psql`.

### 5. Построить отчет

```bash
catalog report --config ./testcases/files/1.yaml --run-id 1 --html-output ./report.html
```

### 6. Сравнить два запуска

```bash
make metadata-diff CATEGORY=mixed CASE=1
```

### 7. Остановить каталог

```bash
make app-down-v
```

## Артефакты релиза

- `metadata-parser_<os>_<arch>.tar.gz`
  архив с бинарем `catalog`
- `metadata-parser-demo.tar.gz`
  demo-bundle с `docker-compose.yml`, `demo/`, `testcases/`, `scripts/`, SQL-миграциями, `Makefile`, `README.md`, `.env`

Скрипты установки:
- [`scripts/install_release.sh`](scripts/install_release.sh) — ставит CLI
- [`scripts/install_demo_bundle.sh`](scripts/install_demo_bundle.sh) — скачивает demo-bundle в `~/metadata-parser-demo`

Go для обычного использования проекта не нужен.

## Формат YAML-конфига

Базовая структура конфига:

```yaml
version: 1

sources:
  - name: source_name
    kind: files
    config:
      path: ./demo/files/test_1
      max_depth: 0
```

Обязательные верхнеуровневые поля:
- `version`
- `sources`

Поддерживаемые `kind`:
- `files`
- `postgres`
- `rest`

Общие поля:
- `version`
  - версия формата конфига
- `sources[].name`
  - имя источника в каталоге и отчетах
- `sources[].kind`
  - тип источника: `files`, `postgres`, `rest`

### Files

```yaml
version: 1

sources:
  - name: files_case
    kind: files
    config:
      path: ./demo/files/test_1
      max_depth: 1
```

Поля:
- `config.path`
  - путь к одному файлу или к директории
- `config.max_depth`
  - максимальная глубина обхода директорий
  - `0` означает только текущую директорию

### PostgreSQL

```yaml
version: 1

sources:
  - name: pg_case
    kind: postgres
    config:
      dsn_env: DEMO_PG_DSN
```

Поля:
- `config.dsn_env`
  - имя env-переменной с DSN PostgreSQL-источника

### REST / OpenAPI

```yaml
version: 1

sources:
  - name: api_case
    kind: rest
    config:
      base_url: http://localhost:8081
      discovery:
        mode: openapi
        openapi_url: http://localhost:8081/openapi.json
```

Поля:
- `config.base_url`
  - базовый URL API, например `http://localhost:8081`
- `config.discovery.mode`
  - способ получения схемы API
  - сейчас поддерживается только `openapi`
- `config.discovery.openapi_url`
  - URL OpenAPI JSON-спеки

Важно:
- текущий REST-коннектор работает только через OpenAPI JSON / Swagger spec
- API должен публиковать такую спецификацию сам
- если у API есть только HTML-документация, Postman collection или вообще нет формальной схемы, этот коннектор сейчас не подойдет

Готовые примеры лежат в:
- [`testcases/files`](testcases/files)
- [`testcases/postgres`](testcases/postgres)
- [`testcases/api`](testcases/api)
- [`testcases/mixed`](testcases/mixed)

Если нужно использовать свои подключения, меняй значения в `.env`, а не имена переменных в самих demo-конфигах.

## Структура demo-данных

Файловые demo-данные лежат в [`demo/files`](demo/files)
и организованы как набор кейсов `test_n`.

Основная схема:
- `demo/files/test_n` — данные файловых источников
- `demo/files/diff/<case>/{baseline,changed}` — состояния для diff по файловым источникам
- `demo/postgres/diff/<case>/{baseline,changed}` — SQL-фикстуры для diff по PostgreSQL-источникам
- `demo/api/diff/<slot>/{baseline,changed}` — OpenAPI и ответы для diff по REST-источникам
- [`testcases/files`](testcases/files) — конфиги файловых кейсов
- [`testcases/postgres`](testcases/postgres) — конфиги PostgreSQL-кейсов
- [`testcases/api`](testcases/api) — конфиги REST/OpenAPI-кейсов
- [`testcases/mixed`](testcases/mixed) — смешанные конфиги
- [`testcases/diff_files`](testcases/diff_files) — файловые diff-кейсы
- [`testcases/diff_postgres`](testcases/diff_postgres) — PostgreSQL diff-кейсы
- [`testcases/diff_api`](testcases/diff_api) — REST diff-кейсы
- [`testcases/diff_mixed`](testcases/diff_mixed) — конфиги под diff-сценарии

Примеры запуска:

```bash
make metadata CATEGORY=files CASE=2
make metadata CATEGORY=postgres CASE=1
make metadata CATEGORY=mixed CASE=1
make metadata-diff CATEGORY=files CASE=1
make metadata-diff CATEGORY=postgres CASE=1
make metadata-diff CATEGORY=api CASE=1
make metadata-diff CATEGORY=mixed CASE=1
catalog run --config ./testcases/files/18.yaml
```

Для категорий `postgres`, `mixed` и `diff_mixed` команда `make metadata`
автоматически подставляет `DEMO_PG_DSN` в базу `source_case_<CASE>` на `localhost:55433`.

## Расширенный demo/e2e

Для сценариев с `postgres`, `files` и `rest` источниками нужен расширенный
стенд из [`demo/postgres/docker-compose.yml`](demo/postgres/docker-compose.yml)
и [`demo/api/docker-compose.yml`](demo/api/docker-compose.yml):

```bash
make app-up
make metadata CATEGORY=postgres CASE=1
make metadata CATEGORY=api CASE=1
make metadata CATEGORY=mixed CASE=1
make metadata-diff CATEGORY=mixed CASE=1
make app-down-v
```

## Diff-сценарии

Команда `make metadata-diff` прогоняет два запуска подряд:
- `baseline`
- `changed`

После этого она сразу вызывает `catalog diff` между двумя полученными `run`.

Поддерживаются категории:
- `files`
- `postgres`
- `api`
- `mixed`

Примеры:

```bash
make metadata-diff CATEGORY=files CASE=1
make metadata-diff CATEGORY=postgres CASE=1
make metadata-diff CATEGORY=api CASE=1
make metadata-diff CATEGORY=mixed CASE=1
```

Что делает команда:
- для `files` подменяет данные из `demo/files/diff/<case>/{baseline,changed}`
- для `postgres` поднимает demo PostgreSQL source в режиме `baseline`, потом в режиме `changed`
- для `api` поднимает нужный demo API-слот в режиме `baseline`, потом в режиме `changed`
- каталоговая БД приложения при этом не удаляется

Скрипт orchestration для diff лежит в [`scripts/metadata_diff.sh`](scripts/metadata_diff.sh).

## Тесты

Запуск всех тестов:

```bash
go test ./...
```

Или через Makefile:

```bash
make test
```

`make test`, `make cover` и `make build` требуют установленный Go и относятся к локальной разработке, а не к обычному пользовательскому запуску.

## Автоматическая e2e-проверка кейсов

Для полного прогона demo-кейсов есть скрипт [`scripts/verify_testcases.sh`](scripts/verify_testcases.sh).

Запуск:

```bash
bash ./scripts/verify_testcases.sh
```

Или по категориям:

```bash
bash ./scripts/verify_testcases.sh files
bash ./scripts/verify_testcases.sh postgres
bash ./scripts/verify_testcases.sh api
bash ./scripts/verify_testcases.sh mixed
bash ./scripts/verify_testcases.sh diff_files diff_postgres diff_api diff_mixed
```

Скрипт для каждого кейса:
- поднимает чистую каталоговую БД
- запускает нужную команду `make metadata ...` или `make metadata-diff ...`
- проверяет, что данные сохранились в каталог
- очищает окружение и переходит к следующему кейсу

Что он проверяет в БД:
- точное ожидаемое количество `runs`
- точное ожидаемое количество `run_sources`
- точный ожидаемый `run status` (`success`, `failed`, `partial`)
- ожидаемый `exit code` команды кейса
- для успешных кейсов: что `datasets` сохранились и их количество не равно `0`
- для полностью провальных кейсов: что `datasets` не появились
- наличие `columns`, `column_stats`, `column_top_values` там, где они должны быть
- корректные `dataset.kind`, `profile_status`, `normalized_type`
- отсутствие зависших `running`
- согласованность статистики:
  - `non_null_count + null_count = row_count`
  - `distinct_count <= non_null_count`
  - `top values` не превышают `non_null_count`
  - ранги и counts у `top values` валидны

Скрипт не сравнивает каждый кейс с полным эталоном вида:
- `datasets = 8`
- `columns = 42`
- конкретный набор имен колонок и точных значений статистики для всех кейсов

То есть это e2e-проверка корректного сохранения и согласованности каталога, а не полный `golden`-oracle на содержимое каждого testcase.

Логи и summary сохраняются в [`tmp/e2e-verify`](tmp/e2e-verify).

Для точечной содержательной проверки отдельных кейсов можно дополнительно:
- запускать конкретный `make metadata CATEGORY=... CASE=...`
- строить `report`
- смотреть сохраненные значения напрямую через `psql`
