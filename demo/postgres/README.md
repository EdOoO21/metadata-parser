Здесь лежат demo-фикстуры PostgreSQL.

Правила:
- `init/001_init_cases.sh` при первом старте контейнера создает базы `source_case_n`
- `test_n/init.sql` содержит SQL-фикстуру для логического кейса `n`
- `diff/<n>/baseline/init.sql` и `diff/<n>/changed/init.sql` содержат два состояния базы `source_case_n` для сценария `metadata-diff`
- `make metadata CATEGORY=postgres CASE=n` использует базу `source_case_n` на `localhost:55433`
- `make metadata CATEGORY=mixed CASE=n` и `diff_mixed` используют ту же схему маппинга
- `make metadata-diff CATEGORY=postgres CASE=n` дважды поднимает тот же `source_case_n`, но с разными diff-фикстурами

Пример:
- `demo/postgres/test_1/init.sql` -> `source_case_1`

Чтобы пересоздать все demo-базы источников с нуля, нужно пересоздать volume demo Postgres:
- `make demo-down-v`
- `make demo-up`

Подготовленные кейсы:
- `test_1` — базовые customers table и view
- `test_2` — sales orders
- `test_3` — hr departments, employees и связанные сущности
- `test_4` — warehouse products и stock movements
- `test_5` — billing invoices, payments и связанные сущности
- `test_6` — support tickets
- `test_7` — поля, похожие на персональные данные
- `test_8` — analytics sessions, events и profiles
- `test_9` — education courses, enrollments и связанные сущности
- `test_10` — marketing leads с большим числом nullable-полей
