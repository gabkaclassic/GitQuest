# GitQuest

[![CI](https://github.com/gabkaclassic/GitQuest/actions/workflows/ci.yml/badge.svg?branch=dev)](https://github.com/gabkaclassic/GitQuest/actions/workflows/ci.yml)
![Go](https://img.shields.io/github/go-mod/go-version/gabkaclassic/GitQuest/dev)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

[English](README.md) | Русский

GitQuest – сервис геймификации для команд разработки. Он принимает события активности (коммиты, смерженные MR, ревью, закрытые issue), сверяет их с правилами из YAML-файла, выдаёт ачивки с очками и отправляет каждую новую ачивку на вебхук.

С GitHub и GitLab сервис сам не общается. События в формат GitQuest переводит CI-джоба, прокси для вебхуков или любой скрипт и отправляет их по HTTP.

## Как это работает

```mermaid
flowchart LR
    client[Прокси вебхуков / CI] -->|POST /event| api[HTTP API]
    api --> pg[(PostgreSQL)]
    api --> redis[(Redis)]
    job[Джоба расчёта] -->|события по пользователю и правилу| redis
    job -->|новые ачивки| pg
    job -->|POST JSON| hook[Вебхук уведомлений]
    reader[Клиент] -->|GET /achievement/user| api
```

1. `POST /event` сохраняет события в PostgreSQL и индексирует их время в Redis: один sorted set на пару «пользователь + тип события».
2. Раз в `CALCULATE_INTERVAL` секунд фоновая джоба обходит всех известных пользователей и для каждого правила считает события нужного типа внутри окна правила.
3. Если число событий подходит под условие правила и за этот период такой ачивки у пользователя ещё нет, джоба сохраняет ачивку и отправляет уведомление на `NOTIFICATION_URL`.
4. Вторая джоба раз в `CLEANUP_INTERVAL` секунд удаляет из Redis события старше 30 дней.

При старте сервис накатывает миграции, читает файл правил и сохраняет каждое правило с версией (SHA-256 от его полей). Если в конфиге `reeval: true` и награда за правило выросла с прошлой версии, пользователи, у которых эта ачивка уже есть, получают разницу и уведомление.

## Быстрый старт

Нужен Docker с Compose. Go 1.24+ понадобится, только чтобы собирать локально и выпустить тестовый токен.

```sh
git clone https://github.com/gabkaclassic/GitQuest.git
cd GitQuest
cp .env.example .env
docker compose up -d --build
```

API слушает `localhost:8080`. Запросам нужен JWT, подписанный `JWT_SECRET` (HS256). Для локальной проверки `cmd/token` выпускает токен на час:

```sh
TOKEN=$(go run ./cmd/token change-me alice)

curl -X POST localhost:8080/event \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "[{\"id\":\"t1\",\"actor\":\"alice\",\"timestamp\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",\"eventType\":\"tag_created\"}]"

curl localhost:8080/achievement/alice \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"
```

Свежее событие `TAG_CREATED` подходит под правило `TagWizard` из примера. Ачивки появляются после следующего прогона джобы расчёта (по умолчанию раз в 10 секунд), поэтому второй запрос вернёт `TagWizard` на 60 очков. Остановить стек: `docker compose down -v`.

## API

Каждый эндпоинт требует `Authorization: Bearer <JWT>`. Если `JWT_SECRET` пустой, аутентификация выключена. Запросы ограничены по IP клиента: `RATELIMIT` запросов за `RATELIMIT_WINDOW` секунд. Все эндпоинты, кроме `/ping`, требуют `Content-Type: application/json`, в том числе GET, и принимают тело в gzip.

| Метод | Путь | Что делает |
|---|---|---|
| `GET` | `/ping` | Проверка живости, `200` с пустым телом |
| `POST` | `/event` | Сохраняет пачку событий |
| `GET` | `/rule` | Возвращает активные правила |
| `GET` | `/achievement/{user}` | Возвращает ачивки пользователя и сумму очков |

`POST /event` принимает JSON-массив:

```json
[
  {"id": "a1b2c3", "actor": "alice", "timestamp": "2025-12-01T10:00:00Z", "eventType": "commit"}
]
```

`id` уникален среди всех событий, `actor` – пользователь, которому засчитывается событие. `eventType` не зависит от регистра и принимает значения `COMMIT`, `MR_OPENED`, `MR_MERGED`, `MR_CLOSED`, `ISSUE_OPENED`, `ISSUE_CLOSED`, `REVIEW_SUBMITTED`, `COMMENT_ADDED`, `TAG_CREATED`, `RELEASE_PUBLISHED`, `REPO_STAR`, `REPO_FORK`.

Ответ `GET /achievement/{user}`:

```json
{"achievements": [{"rule": "EarlyBird", "reward": 50}], "rewardSum": 50}
```

`GET /rule` отдаёт правила из конфига, `window` в наносекундах.

Ошибки приходят как `{"error": "<сообщение>"}`: `400` – неверный Content-Type, `401` – невалидный токен, `403` – токена нет, `422` – битый JSON, `429` – превышен лимит запросов.

### Вебхук уведомлений

На каждую пачку новых ачивок сервис шлёт `POST NOTIFICATION_URL` с заголовком `NOTIFICATION_HEADER: NOTIFICATION_HEADER_VALUE` и таким телом:

```json
[
  {"id": "6f1c...", "user": "alice", "reward": 50, "achievement": "EarlyBird", "description": "5 коммитов за последние 24 часа"}
]
```

## Правила

Правила лежат в `config/rules.yaml` (путь задаёт флаг `-r`):

```yaml
reeval: true

rules:
  - name: "EarlyBird"
    description: "5 коммитов за последние 24 часа"
    event_type: "commit"
    window: 24h
    count: 5
    condition: ">="
    streak: false
    reward: 50
```

| Поле | Значение |
|---|---|
| `name` | Уникальное имя правила, его видно в ачивках |
| `description` | Текст для уведомления |
| `event_type` | Один из типов событий выше |
| `window` | Длительность в формате Go, до `720h`. `0s` – без окна: считаются все события, которые ещё лежат в Redis |
| `count` | Порог для `condition`. Для streak-правила ещё и число дней подряд |
| `condition` | `>`, `>=`, `==`, `<` или `<=`, проверяется как `событий_в_окне <condition> count` |
| `streak` | Если `true`, нужно хотя бы одно событие в каждый из последних `count` дней (UTC); `window` должен покрывать эти дни |
| `reward` | Очки за ачивку |

Изменение любого поля, кроме `name` и `description`, создаёт новую версию правила.

## Конфигурация

Любую настройку можно задать переменной окружения или флагом; если заданы оба, побеждает флаг. Длительности – целые числа в секундах.

| Переменная | Флаг | По умолчанию | Назначение |
|---|---|---|---|
| `ADDRESS` | `-a` | `localhost:8080` | Адрес HTTP-сервера |
| `JWT_SECRET` | `-jwt-secret` | пусто | Секрет HS256; пустой выключает аутентификацию |
| `RATELIMIT` | `-ratelimit` | `600` | Запросов за окно с одного IP |
| `RATELIMIT_WINDOW` | `-ratelimit-window` | `60` | Окно лимита |
| `DATABASE_DSN` | `-d` | нет, обязательно | DSN PostgreSQL |
| `DB_DRIVER` | `-db-driver` | `postgres` | Драйвер `database/sql` |
| `DB_MIGRATIONS_PATH` | `-db-migrations-path` | `./migrations` | Каталог миграций, накатываются при старте |
| `CACHE_ADDRESS` | `-cache-addr` | `localhost:6379` | Адрес Redis |
| `CACHE_USER` | `-cache-user` | пусто | Пользователь Redis |
| `CACHE_PASSWORD` | `-cache-pass` | пусто | Пароль Redis |
| `CACHE_DB` | `-cache-db` | `0` | Номер базы Redis |
| `LOG_LEVEL` | `-log-level` | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_FILE` | `-log-file` | пусто | Файл логов |
| `LOG_CONSOLE` | `-log-console` | `false` | Писать логи в stdout |
| `LOG_JSON` | `-log-json` | `true` | Логи в JSON |
| `CALCULATE_INTERVAL` | `-calculate-interval` | `10` | Период расчёта ачивок |
| `CALCULATE_TIMEOUT` | `-calculate-timeout` | `10` | Таймаут одного расчёта |
| `CLEANUP_INTERVAL` | `-cleanup-interval` | `10` | Период очистки Redis |
| `CLEANUP_TIMEOUT` | `-cleanup-timeout` | `10` | Таймаут одной очистки |
| `NOTIFICATION_URL` | `-notification-url` | пусто | Вебхук для новых ачивок |
| `NOTIFICATION_HEADER` | `-notification-header` | `Authorization` | Заголовок для вебхука |
| `NOTIFICATION_HEADER_VALUE` | `-notification-header-val` | пусто | Значение этого заголовка |
| | `-r` | `./config/rules.yaml` | Файл правил |

## Структура проекта

```
cmd/server        точка входа сервиса
cmd/token         утилита для тестового JWT
config/           пример правил
internal/app      сборка зависимостей и жизненный цикл
internal/handler  HTTP-хендлеры и роутер
internal/service  бизнес-логика: события, правила, ачивки, уведомления
internal/job      фоновые джобы расчёта и очистки
internal/cache    работа с Redis
internal/repository  работа с PostgreSQL
migrations/       SQL-схема (golang-migrate)
pkg/middleware    JWT-миддлварь
```

## Разработка

```sh
make hooks   # включить git-хуки: gofmt при коммите, тесты при пуше
make build   # бинарники в ./build
make test    # тесты с отчётом о покрытии
make lint    # go vet и проверка gofmt
make mocks   # перегенерировать моки (mockery v3)
```

Чтобы запустить сервис без Docker, поднимите только зависимости (`docker compose up -d postgres redis`) и выполните:

```sh
DATABASE_DSN='postgres://gitquest-user:password@localhost:5432/gitquest?sslmode=disable' \
  LOG_CONSOLE=true JWT_SECRET=change-me go run ./cmd/server
```

Новая миграция: `make migrate-create` (нужен [migrate CLI](https://github.com/golang-migrate/migrate)).

## Известные ограничения

- Redis хранит события за 30 дней, поэтому окно `0s` считает последние 30 дней, а не всю историю.

## Лицензия

[MIT](LICENSE)
