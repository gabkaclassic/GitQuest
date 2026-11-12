# GitQuest

[![CI](https://github.com/gabkaclassic/GitQuest/actions/workflows/ci.yml/badge.svg?branch=dev)](https://github.com/gabkaclassic/GitQuest/actions/workflows/ci.yml)
![Go](https://img.shields.io/github/go-mod/go-version/gabkaclassic/GitQuest/dev)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

English | [Русский](README.ru.md)

GitQuest is a gamification service for development teams. You send it activity events (commits, merged MRs, reviews, closed issues), it checks them against rules from a YAML file, grants achievements with points and posts each new achievement to a webhook.

The service does not talk to GitHub or GitLab itself. A CI job, a webhook relay or any script converts repository activity into GitQuest events and posts them over HTTP.

## How it works

```mermaid
flowchart LR
    client[Webhook relay / CI] -->|POST /event| api[HTTP API]
    api --> pg[(PostgreSQL)]
    api --> redis[(Redis)]
    job[Calculate job] -->|events per user and rule| redis
    job -->|new achievements| pg
    job -->|POST JSON| hook[Notification webhook]
    reader[Client] -->|GET /achievement/user| api
```

1. `POST /event` stores events in PostgreSQL and indexes their timestamps in Redis, one sorted set per user and event type.
2. Every `CALCULATE_INTERVAL` seconds a background job walks all known users and, for each rule, counts that user's events of the rule's type inside the rule's time window.
3. When the count satisfies the rule's condition and the user has no such achievement for that period yet, the job saves the achievement and sends a notification to `NOTIFICATION_URL`.
4. A second job, every `CLEANUP_INTERVAL` seconds, drops event timestamps older than 30 days from Redis.

On startup the service applies database migrations, loads the rules file and stores every rule with a version (a SHA-256 of its fields). If `reeval: true` is set and a rule's reward has grown since the stored version, users who already hold that achievement get the difference and a notification.

## Quick start

Requirements: Docker with Compose. Go 1.24+ is only needed to build locally and to mint a test token.

```sh
git clone https://github.com/gabkaclassic/GitQuest.git
cd GitQuest
cp .env.example .env
docker compose up -d --build
```

The API listens on `localhost:8080`. Requests need a JWT signed with `JWT_SECRET` (HS256). For local testing, `cmd/token` issues one valid for an hour:

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

A fresh `TAG_CREATED` event matches the example `TagWizard` rule. Achievements appear after the next run of the calculate job (10 seconds by default), so the second request returns `TagWizard` with 60 points. Stop the stack with `docker compose down -v`.

## API

Every endpoint requires `Authorization: Bearer <JWT>` unless `JWT_SECRET` is empty, in which case authentication is off. Requests are rate-limited per client IP (`RATELIMIT` requests per `RATELIMIT_WINDOW` seconds). Endpoints other than `/ping` require `Content-Type: application/json`, including GET requests, and accept gzip-compressed bodies.

| Method | Path | Description |
|---|---|---|
| `GET` | `/ping` | Liveness check, returns `200` with an empty body |
| `POST` | `/event` | Stores a batch of events |
| `GET` | `/rule` | Returns the active rules |
| `GET` | `/achievement/{user}` | Returns a user's achievements and total points |

`POST /event` takes a JSON array:

```json
[
  {"id": "a1b2c3", "actor": "alice", "timestamp": "2025-12-01T10:00:00Z", "eventType": "commit"}
]
```

`id` must be unique across all events; `actor` is the user the event counts for. `eventType` is case-insensitive and is one of `COMMIT`, `MR_OPENED`, `MR_MERGED`, `MR_CLOSED`, `ISSUE_OPENED`, `ISSUE_CLOSED`, `REVIEW_SUBMITTED`, `COMMENT_ADDED`, `TAG_CREATED`, `RELEASE_PUBLISHED`, `REPO_STAR`, `REPO_FORK`.

`GET /achievement/{user}` response:

```json
{"achievements": [{"rule": "EarlyBird", "reward": 50}], "rewardSum": 50}
```

`GET /rule` returns the rules from the config. `window` is in nanoseconds.

Errors come back as `{"error": "<message>"}`: `400` for a wrong content type, `401` for an invalid token, `403` for a missing token, `422` for malformed JSON, `429` when the rate limit is hit.

### Notification webhook

For every batch of new achievements the service sends `POST NOTIFICATION_URL` with the header `NOTIFICATION_HEADER: NOTIFICATION_HEADER_VALUE` and this body:

```json
[
  {"id": "6f1c...", "user": "alice", "reward": 50, "achievement": "EarlyBird", "description": "5 commits in the last 24 hours"}
]
```

## Rules

Rules live in `config/rules.yaml` (path set by the `-r` flag):

```yaml
reeval: true

rules:
  - name: "EarlyBird"
    description: "5 commits in the last 24 hours"
    event_type: "commit"
    window: 24h
    count: 5
    condition: ">="
    streak: false
    reward: 50
```

| Field | Meaning |
|---|---|
| `name` | Unique rule name, shown in achievements |
| `description` | Text passed to the notification |
| `event_type` | One of the event types above |
| `window` | Go duration, up to `720h`. `0s` means no window: every event still stored in Redis counts |
| `count` | Threshold for `condition`. For a streak rule, also the number of consecutive days |
| `condition` | `>`, `>=`, `==`, `<` or `<=`, applied as `events_in_window <condition> count` |
| `streak` | When `true`, the user needs at least one event on each of the last `count` days (UTC); `window` must cover those days |
| `reward` | Points for the achievement |

Changing any field except `name` and `description` creates a new rule version.

## Configuration

Every setting can come from an environment variable or a command-line flag; a flag wins when both are set. Durations are integers in seconds.

| Variable | Flag | Default | Purpose |
|---|---|---|---|
| `ADDRESS` | `-a` | `localhost:8080` | HTTP listen address |
| `JWT_SECRET` | `-jwt-secret` | empty | HS256 secret; empty disables auth |
| `RATELIMIT` | `-ratelimit` | `600` | Requests per window per IP |
| `RATELIMIT_WINDOW` | `-ratelimit-window` | `60` | Rate limit window |
| `DATABASE_DSN` | `-d` | none, required | PostgreSQL DSN |
| `DB_DRIVER` | `-db-driver` | `postgres` | `database/sql` driver |
| `DB_MIGRATIONS_PATH` | `-db-migrations-path` | `./migrations` | Migrations directory, applied on start |
| `CACHE_ADDRESS` | `-cache-addr` | `localhost:6379` | Redis address |
| `CACHE_USER` | `-cache-user` | empty | Redis username |
| `CACHE_PASSWORD` | `-cache-pass` | empty | Redis password |
| `CACHE_DB` | `-cache-db` | `0` | Redis database |
| `LOG_LEVEL` | `-log-level` | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_FILE` | `-log-file` | empty | Log file path |
| `LOG_CONSOLE` | `-log-console` | `false` | Log to stdout |
| `LOG_JSON` | `-log-json` | `true` | JSON log format |
| `CALCULATE_INTERVAL` | `-calculate-interval` | `10` | Achievement calculation period |
| `CALCULATE_TIMEOUT` | `-calculate-timeout` | `10` | Timeout of one calculation run |
| `CLEANUP_INTERVAL` | `-cleanup-interval` | `10` | Redis cleanup period |
| `CLEANUP_TIMEOUT` | `-cleanup-timeout` | `10` | Timeout of one cleanup run |
| `NOTIFICATION_URL` | `-notification-url` | empty | Webhook for new achievements |
| `NOTIFICATION_HEADER` | `-notification-header` | `Authorization` | Header sent to the webhook |
| `NOTIFICATION_HEADER_VALUE` | `-notification-header-val` | empty | Value of that header |
| | `-r` | `./config/rules.yaml` | Rules file |

## Project layout

```
cmd/server        service entry point
cmd/token         helper that issues a test JWT
config/           example rules
internal/app      wiring and lifecycle
internal/handler  HTTP handlers and router
internal/service  business logic: events, rules, achievements, notifications
internal/job      background calculate and cleanup jobs
internal/cache    Redis access
internal/repository  PostgreSQL access
migrations/       SQL schema (golang-migrate)
pkg/middleware    JWT middleware
```

## Development

```sh
make hooks   # enable git hooks: gofmt on commit, tests on push
make build   # binaries in ./build
make test    # tests with a coverage report
make lint    # go vet and gofmt check
make mocks   # regenerate mocks with mockery
```

To run the service outside Docker, start only the dependencies with `docker compose up -d postgres redis`, then:

```sh
DATABASE_DSN='postgres://gitquest-user:password@localhost:5432/gitquest?sslmode=disable' \
  LOG_CONSOLE=true JWT_SECRET=change-me go run ./cmd/server
```

New migrations: `make migrate-create` (needs the [migrate CLI](https://github.com/golang-migrate/migrate)).

## Known limitations

- Redis keeps 30 days of events, so a `0s` window counts the last 30 days, not all history.
- `GET /achievement/{user}` caches the response for 5 minutes, and new achievements do not reset that cache.

## License

[MIT](LICENSE)
