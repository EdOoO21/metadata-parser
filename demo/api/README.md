Здесь лежат demo-фикстуры REST API и отдельный compose для их запуска.

Правила:
- `test_<n>/openapi.json` описывает OpenAPI-схему для API-слота `n`
- `test_<n>/routes.json` содержит JSON-ответы, разложенные по путям
- `diff/<n>/baseline` и `diff/<n>/changed` содержат два состояния одного и того же API-слота для сценария `metadata-diff`
- API-слоты во время запуска фиксированы:
  - `demo_api_1` -> `localhost:8081` -> `test_1`
  - `demo_api_2` -> `localhost:8082` -> `test_2`
  - `demo_api_3` -> `localhost:8083` -> `test_3`
  - `demo_api_4` -> `localhost:8084` -> `test_4`
  - `demo_api_5` -> `localhost:8085` -> `test_5`
  - `demo_api_6` -> `localhost:8086` -> `test_6`
  - `demo_api_7` -> `localhost:8087` -> `test_7`
  - `demo_api_8` -> `localhost:8088` -> `test_8`
  - `demo_api_9` -> `localhost:8089` -> `test_9`
- [`testcases/api`](../../testcases/api) комбинирует эти слоты в разных наборах
- [`testcases/mixed`](../../testcases/mixed) и [`testcases/diff_mixed`](../../testcases/diff_mixed) используют выбранные API-слоты повторно
- [`testcases/diff_api`](../../testcases/diff_api) прогоняет baseline/changed по одному фиксированному слоту

Подготовленные фикстуры:
- `1` — 2 ручки: users и orders
- `2` — 5 ручек: customers, products, orders, payments, cities
- `3` — 4 ручки с потенциально чувствительными полями: people, documents, contacts, profiles
- `4` — 6 operational-ручек: sessions, events, devices, metrics, regions, alerts
- `5` — 3 ручки с object- и array-ответами: summary, tasks, health
- `6` — 10 ручек со смешанными бизнес-данными
- `7` — catalog и warehouse ручки с path-параметрами `sku` и `id`
- `8` — customer и support ручки с вложенными path-параметрами
- `9` — OpenAPI с разными HTTP-методами: GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS, TRACE
