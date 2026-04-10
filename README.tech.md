# Marketplace Tech README

Техническая документация по устройству, локальному запуску, сопровождению и развитию проекта `marketplace`.

> Этот файл опирается на текущую структуру репозитория, `docker-compose`-конфиги, `Makefile`, `scripts/`, OpenAPI и код backend/frontend. Если какого-то факта нельзя уверенно подтвердить по репозиторию, он помечен как `[нужно уточнить]`.

## Технический обзор

Проект собран как full-stack monorepo:

- backend на Go 1.24 с HTTP API на Chi;
- PostgreSQL как основная база данных;
- миграции на Goose;
- frontend на React + Vite + TypeScript;
- Docker Compose для локальной среды;
- Prometheus/Grafana для метрик и дашбордов;
- OpenAPI-спека и Swagger UI из самого сервиса;
- фоновые задачи внутри backend-процесса.

В текущем виде это не только “витрина каталога”, а полноценный каркас marketplace-системы с buyer-, seller- и admin-сценариями, сессиями, защитой auth-потоков, рекомендациями и эксплуатационными скриптами.

## Архитектура проекта

| Слой | Что делает | Ключевые детали |
| --- | --- | --- |
| Frontend | Показывает пользовательские и служебные интерфейсы | React 19, Vite 7, TypeScript, Redux Toolkit, React Router, Axios, Sass |
| HTTP API | Обрабатывает клиентские и служебные запросы | Chi router, middleware для auth, CSRF, rate limit, security headers, metrics |
| Use cases | Держит бизнес-логику | auth, catalog, cart, orders, favorites, places, profile, recommendations, seller, admin |
| Repository layer | Изолирует доступ к PostgreSQL | pgx/pgxpool, SQL-запросы в `internal/repository/postgres` |
| Background jobs | Выполняет периодические задачи | cleanup, email dispatch, refresh popularity stats, refresh recommendation cache |
| Observability | Даёт метрики и служебные логи | `/healthz`, `/readyz`, `/metrics`, audit/error events, Prometheus/Grafana |

### Поток данных в общем виде

1. Frontend отправляет REST-запросы в `/api/v1/...`.
2. Backend проходит через middleware: security headers, request id, metrics, logging, recoverer, auth, CSRF и rate limits.
3. Use case работает с PostgreSQL-репозиториями и при необходимости пишет события/аудит.
4. Фоновые задачи обновляют служебные данные и обрабатывают email-очередь.
5. Prometheus собирает метрики, Grafana показывает готовые дашборды.

## Структура репозитория

| Путь | Назначение |
| --- | --- |
| `cmd/api/` | Основная точка входа backend API. |
| `cmd/migrate/` | CLI для миграций (`up`, `down`, `status`, `version`) с embedded SQL. |
| `internal/app/` | Сборка приложения и wiring зависимостей. |
| `internal/config/` | Загрузка и валидация env-конфига. |
| `internal/domain/` | Доменные типы и роли. |
| `internal/http/` | Router, handlers, middleware. |
| `internal/usecase/` | Бизнес-логика по модулям. |
| `internal/repository/postgres/` | PostgreSQL-репозитории. |
| `internal/security/` | JWT, password hashing, cookie-auth helpers. |
| `internal/jobs/` | Планировщик и задачи background jobs. |
| `internal/observability/` | Метрики, audit logging, error reporting, DB collector. |
| `internal/mailer/` | Очередь email и логирующий mail/SMS transport. |
| `frontend/` | React/Vite приложение. |
| `apidocs/` | `openapi.yaml` и HTTP-обработчики для документации. |
| `migrations/` | SQL-миграции `00001`...`00020` и `embed.go`. |
| `docker/` | Init SQL, monitoring-конфиги, Grafana dashboards, Alertmanager config. |
| `scripts/` | Dev, QA, deploy, backup, restore, maintenance scripts. |
| `.github/workflows/` | CI/CD/release workflows. |
| `qa-artifacts/` | Артефакты QA-скриптов. |
| `test/` | Каталог присутствует, но его роль в текущем состоянии репозитория явно не раскрыта: [нужно уточнить]. |

## Backend stack

| Компонент | Что используется |
| --- | --- |
| Язык | Go 1.24 (`go.mod`) |
| HTTP router | `github.com/go-chi/chi/v5` |
| PostgreSQL driver/pool | `github.com/jackc/pgx/v5`, `pgxpool` |
| Миграции | Goose (`pressly/goose`) |
| Валидация | `go-playground/validator/v10` |
| JWT | `golang-jwt/jwt/v5` |
| OpenAPI | `kin-openapi`, `swgui` |
| Метрики | `prometheus/client_golang` |
| Тесты | стандартный `testing` + `stretchr/testify` |

## Frontend stack

| Компонент | Что используется |
| --- | --- |
| UI | React 19 |
| Сборка | Vite 7 |
| Язык | TypeScript 5.9 |
| State management | Redux Toolkit + React Redux |
| Роутинг | React Router DOM 6 |
| HTTP client | Axios |
| Стили | Sass |
| Тесты | Vitest + Testing Library + jsdom |
| Форматирование | Prettier |
| Линтинг | ESLint 9 |

## Инфраструктура и observability

| Часть | Что есть в репозитории |
| --- | --- |
| Локальная среда | `docker-compose.yml` поднимает `postgres`, `api`, `adminer`, `prometheus`, `grafana` |
| Production compose | `docker-compose.prod.yml` поднимает `postgres`, `migrate`, `api`, `prometheus`, `alertmanager`, `grafana` |
| База данных | PostgreSQL 16 |
| DB admin UI | Adminer на `localhost:8081` |
| Метрики | Prometheus на `localhost:9090` |
| Дашборды | Grafana на `localhost:3000` |
| Health checks | `/healthz`, `/readyz` |
| Служебные метрики | `/metrics` |
| Audit trail | `audit_logs` + логирование audit-событий |
| Error tracking | `error_events` + error reporter |

### Что мониторится

По коду и конфигам подтверждены:

- HTTP requests in flight, total и latency;
- ошибки приложения;
- audit events;
- метрики пула БД;
- алерты на недоступность API, рост 5xx, p95 latency и насыщение DB pool.

## Основные доменные модули

| Модуль | Что покрывает | Примеры маршрутов |
| --- | --- | --- |
| Auth | регистрация, логин, refresh, verify email, reset password, session inventory | `/api/v1/auth/*` |
| Catalog | категории, список товаров, карточка товара, search hints, popular searches | `/api/v1/categories`, `/api/v1/products`, `/api/v1/search/*` |
| Favorites | избранное пользователя | `/api/v1/favorites` |
| Cart | корзина и позиции | `/api/v1/cart` |
| Places | адреса/места доставки пользователя | `/api/v1/places` |
| Orders | checkout и история заказов | `/api/v1/orders` |
| Profile | профиль пользователя | `/api/v1/profile` |
| Recommendations | выдача рекомендаций | `/api/v1/recommendations` |
| Seller | seller profile, dashboard, товары, seller orders | `/api/v1/seller/*` |
| Admin | управление категориями и товарами | `/api/v1/admin/*` |
| Reviews | список и создание отзывов | `/api/v1/products/{id}/reviews` |

## Auth, sessions и security

### Что поддерживается

- Регистрация по email/password.
- Обязательная email-верификация перед обычным входом.
- Восстановление пароля по одноразовому токену.
- Вход по email-коду и phone-коду.
- Access token + refresh token.
- Инвентарь активных сессий и отзыв конкретной сессии.
- Logout текущей сессии и logout-all.
- Режим bearer auth и отдельный режим cookie auth.

### Как устроена защита

- JWT используются для access token, refresh управляется через серверные сессии.
- При cookie-mode backend выставляет `mp_access_token`, `mp_refresh_token` и `mp_csrf_token`.
- Для небезопасных методов в cookie-mode включён double-submit CSRF-check через cookie и заголовок `X-CSRF-Token`.
- Есть rate limit на регистрацию, логин, refresh, verify email и password reset.
- Есть lockout после серии неудачных попыток логина.
- Middleware добавляет security headers: `X-Frame-Options`, `X-Content-Type-Options`, `Referrer-Policy`, `Permissions-Policy`, `Cross-Origin-Opener-Policy`, HSTS для HTTPS.

> Важная практическая деталь: email и SMS-провайдеры в коде сейчас не подключены. Email уходят в очередь `email_jobs`, а фактическая “доставка” реализована логирующим sender'ом. SMS тоже только логируются.

### Что делает frontend

- В `VITE_AUTH_MODE=token` клиент отправляет bearer access token и сам обновляет его через refresh endpoint.
- В `VITE_AUTH_MODE=cookie` клиент работает с `withCredentials=true` и автоматически прокидывает `X-CSRF-Token` для небезопасных запросов.

## Background jobs

Фоновые задачи запускаются внутри backend-процесса, если `JOBS_ENABLED=true`.

| Задача | Что делает | Управляющие переменные |
| --- | --- | --- |
| Cleanup | удаляет просроченные/отозванные сессии и action tokens, чистит email-очередь | `JOB_CLEANUP_INTERVAL` |
| Email dispatch | забирает pending email jobs, отправляет, ретраит и помечает failed | `JOB_EMAIL_*` |
| Stats refresh | пересчитывает популярность товаров | `JOB_STATS_REFRESH_INTERVAL` |
| Recommendations refresh | обновляет кэш пользовательских рекомендаций | `JOB_RECOMMENDATIONS_*` |

Рекомендации формируются на базе активности пользователя, избранного, заказов и popularity stats; кэш хранится в БД.

## API и OpenAPI

- OpenAPI-спека: `apidocs/openapi.yaml`
- Swagger UI: `http://localhost:8080/docs/`
- Raw spec: `http://localhost:8080/openapi.yaml`

По спеке и роутеру подтверждены следующие области API:

- Health
- Auth
- Catalog
- Search
- Profile
- Favorites
- Cart
- Places
- Orders
- Seller
- Recommendations
- Admin

Формат ответа унифицирован:

- успешный ответ: `{ "data": ... }`
- ошибка: `{ "error": { "code", "message", "details" } }`

`make openapi-check` запускает тесты из `apidocs/`, которые валидируют спеку.

## Требования к окружению

| Что нужно | Комментарий |
| --- | --- |
| Go 1.24+ | подтверждено `go.mod` |
| Node.js + npm | нужны для frontend; минимальная версия явно не зафиксирована, [нужно уточнить] |
| Docker / Docker Compose | нужны для локальной БД и всей compose-среды |
| PowerShell | для `start-dev.cmd` и `scripts/*.ps1` на Windows |
| Bash | для production/deploy/backup/restore-скриптов |

## Переменные окружения

Основные шаблоны уже лежат в репозитории:

- `./.env.example`

## AI product assistant

- MVP AI-ассистент встроен в существующую seller product form и не создаёт, не сохраняет и не публикует товар автоматически.
- Backend endpoint: `POST /api/v1/seller/ai/product-card`.
- Архитектура модуля повторяет проектный стиль: `internal/usecase/seller_ai.go` валидирует и нормализует входные данные, собирает prompt/schema, вызывает AI generator interface и санитизирует результат; `internal/integrations/openai/client.go` реализует backend-only интеграцию с OpenAI Responses API; `internal/http/handlers/seller_ai.go` обслуживает seller-only HTTP endpoint.
- Structured output ограничен полями текущей product form: `name`, `slug`, `description`, `brand`, `unit`, `specs`, плюс `warnings` и `missing_fields`.
- Если `AI_PRODUCT_ASSISTANT_ENABLED=false` или не настроен `OPENAI_API_KEY`, endpoint возвращает controlled `503`, а seller UI продолжает работать без AI.
- `./.env.production.example`
- `frontend/.env.example`
- `frontend/.env.production.example`

> Дополнительно backend поддерживает `AUTH_LOGIN_CODE_TTL` с дефолтом `10m`, хотя эта переменная не вынесена в `./.env.example`.

### Backend: базовые переменные

| Переменная | Где используется | Пример/дефолт |
| --- | --- | --- |
| `APP_ENV` | режим приложения | `development`, `test`, `production` |
| `HTTP_PORT` | порт API | `8080` |
| `DATABASE_URL` | основная БД | `postgres://.../marketplace` |
| `TEST_DATABASE_URL` | тестовая БД | `postgres://.../marketplace_test` |
| `JWT_SECRET` | подпись JWT, минимум 32 символа | `replace_with_a_long_random_secret...` |
| `APP_BASE_URL` | базовый URL frontend/app links | `http://localhost:5173` |
| `MAIL_FROM` | sender для email | `no-reply@marketplace.local` |
| `ADMIN_EMAILS` | список админских email | `admin@example.com` |
| `LOG_LEVEL` | уровень логирования | `debug`, `info`, `warn`, `error` |
| `HTTP_READ_TIMEOUT` | timeout чтения HTTP | `10s` |
| `HTTP_WRITE_TIMEOUT` | timeout записи HTTP | `15s` |

### Backend: auth и security

| Переменная | Назначение | Пример |
| --- | --- | --- |
| `ACCESS_TOKEN_TTL` | срок жизни access token | `15m` |
| `REFRESH_TOKEN_TTL` | срок жизни refresh token | `720h` |
| `EMAIL_VERIFY_TTL` | срок действия verify token | `24h` |
| `PASSWORD_RESET_TTL` | срок действия reset token | `1h` |
| `AUTH_LOGIN_CODE_TTL` | срок жизни email/phone login code | `10m` |
| `AUTH_LOGIN_FAILURE_WINDOW` | окно подсчёта ошибок логина | `15m` |
| `AUTH_LOGIN_MAX_ATTEMPTS` | число попыток до lockout | `5` |
| `AUTH_LOGIN_LOCKOUT_DURATION` | длительность lockout | `15m` |
| `AUTH_RATE_LIMIT_REGISTER` | лимит регистрации | `5` |
| `AUTH_RATE_LIMIT_REGISTER_WINDOW` | окно лимита регистрации | `1m` |
| `AUTH_RATE_LIMIT_LOGIN` | лимит логина | `10` |
| `AUTH_RATE_LIMIT_LOGIN_WINDOW` | окно лимита логина | `1m` |
| `AUTH_RATE_LIMIT_REFRESH` | лимит refresh | `30` |
| `AUTH_RATE_LIMIT_REFRESH_WINDOW` | окно лимита refresh | `1m` |
| `AUTH_RATE_LIMIT_PASSWORD_RESET` | лимит reset | `5` |
| `AUTH_RATE_LIMIT_PASSWORD_RESET_WINDOW` | окно лимита reset | `15m` |
| `AUTH_RATE_LIMIT_VERIFY_EMAIL` | лимит verify email | `5` |
| `AUTH_RATE_LIMIT_VERIFY_EMAIL_WINDOW` | окно лимита verify email | `15m` |
| `AUTH_COOKIE_MODE` | включает cookie auth | `false` в dev, `true` в prod example |
| `AUTH_COOKIE_SECURE` | secure-флаг auth cookies | `false` / `true` |
| `AUTH_COOKIE_DOMAIN` | домен cookies | пусто / `.example.com` |
| `AUTH_COOKIE_SAME_SITE` | SameSite | `lax`, `strict`, `none` |
| `AUTH_CSRF_ENABLED` | включает CSRF-check | `true` |

### Backend: jobs, monitoring и prod

| Переменная | Назначение | Пример |
| --- | --- | --- |
| `JOBS_ENABLED` | включает background jobs | `true` |
| `JOB_CLEANUP_INTERVAL` | период cleanup | `1h` |
| `JOB_EMAIL_POLL_INTERVAL` | polling email jobs | `5s` |
| `JOB_EMAIL_LOCK_TTL` | ttl блокировки email job | `2m` |
| `JOB_EMAIL_BATCH_SIZE` | размер email batch | `20` |
| `JOB_EMAIL_MAX_ATTEMPTS` | максимум попыток email | `5` |
| `JOB_EMAIL_RETENTION` | хранение обработанных email jobs | `168h` |
| `JOB_STATS_REFRESH_INTERVAL` | refresh popularity stats | `10m` |
| `JOB_RECOMMENDATIONS_REFRESH_INTERVAL` | refresh recommendation cache | `15m` |
| `JOB_RECOMMENDATION_ACTIVITY_WINDOW` | окно активности для рекомендаций | `168h` |
| `JOB_RECOMMENDATION_USER_BATCH_SIZE` | размер user batch | `200` |
| `JOB_RECOMMENDATION_LIMIT` | число рекомендаций на пользователя | `20` |
| `GRAFANA_ADMIN_USER` | логин Grafana | `admin` |
| `GRAFANA_ADMIN_PASSWORD` | пароль Grafana | `admin` / production secret |
| `GRAFANA_ROOT_URL` | public root URL Grafana | `https://grafana.example.com` |
| `ALERTMANAGER_EXTERNAL_URL` | внешний URL Alertmanager | `https://alerts.example.com` |
| `API_IMAGE` | образ API для prod compose | `ghcr.io/...` |
| `POSTGRES_DB` | имя prod БД | `marketplace` |
| `POSTGRES_USER` | пользователь prod БД | `marketplace` |
| `POSTGRES_PASSWORD` | пароль prod БД | `change_me_strong_password` |
| `API_BIND_PORT` | bind port API в prod compose | `8080` |
| `PROMETHEUS_BIND_PORT` | bind port Prometheus | `9090` |
| `ALERTMANAGER_BIND_PORT` | bind port Alertmanager | `9093` |
| `GRAFANA_BIND_PORT` | bind port Grafana | `3000` |
| `BACKUP_DIR` | каталог dump-файлов | `/opt/marketplace/backups` |
| `BACKUP_RETENTION_DAYS` | срок хранения backup'ов | `14` |

### Frontend env

| Переменная | Назначение | Пример |
| --- | --- | --- |
| `VITE_API_BASE_URL` | базовый URL API для клиента | `/api` или `https://api.example.com/api` |
| `VITE_API_PROXY_TARGET` | proxy target для Vite dev server | `http://localhost:8080` |
| `VITE_AUTH_MODE` | режим auth клиента | `token` или `cookie` |

## Локальный запуск через Docker Compose

### Рекомендуемый способ для Windows

```powershell
Copy-Item .env.example .env
Copy-Item frontend/.env.example frontend/.env
.\scripts\dev-all.ps1 -OpenBrowser
```

Что делает `scripts/dev-all.ps1`:

- создаёт отсутствующие `.env` из примеров;
- поднимает `postgres`, затем `api`, `adminer`, `prometheus`, `grafana`;
- прогоняет миграции через контейнер;
- ставит frontend-зависимости при необходимости;
- стартует Vite dev server.

Альтернатива: `.\start-dev.cmd`, который просто вызывает этот же PowerShell-скрипт с `-OpenBrowser`.

### Ручной compose-запуск

```powershell
Copy-Item .env.example .env
docker compose up -d --build postgres
docker compose run --rm api /app/migrate up
docker compose up -d --build api adminer prometheus grafana
```

Frontend после этого запускается отдельно:

```powershell
Copy-Item frontend/.env.example frontend/.env
cd frontend
npm install
npm run dev
```

### Локальные URL

| Сервис | URL |
| --- | --- |
| Frontend | `http://localhost:5173` |
| Backend API | `http://localhost:8080` |
| Swagger UI | `http://localhost:8080/docs/` |
| OpenAPI spec | `http://localhost:8080/openapi.yaml` |
| Adminer | `http://localhost:8081` |
| Prometheus | `http://localhost:9090` |
| Grafana | `http://localhost:3000` |
| PostgreSQL с хоста | `localhost:5433` |

> Локальный `docker-compose.yml` также монтирует `docker/initdb/01-create-test-db.sql`, чтобы создать `marketplace_test` для тестов.

## Локальный запуск backend и frontend по отдельности

### Backend без полного compose-стека

Если нужна только база в Docker, а API хочется гонять локально:

```powershell
Copy-Item .env.example .env
docker compose up -d postgres
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5433/marketplace?sslmode=disable"
go run ./cmd/api
```

Готовые скрипты:

- `scripts/dev-backend.ps1` запускает `go run ./cmd/api`
- `scripts/start-qa-api.ps1` поднимает backend на `http://localhost:18080` с локальной БД и `AUTH_COOKIE_MODE=false`

### Frontend отдельно

```powershell
Copy-Item frontend/.env.example frontend/.env
cd frontend
npm install
npm run dev
```

Vite проксирует `/api`, `/healthz` и `/readyz` на `VITE_API_PROXY_TARGET` или по умолчанию на `http://localhost:8080`.

## Миграции

В проекте есть два практических способа работать с миграциями.

### Через Makefile и Goose

```bash
make migrate-up
make migrate-down
```

`make seed` сейчас эквивалентен `make migrate-up`: отдельного seed-скрипта нет, сиды зашиты в миграции.

### Через встроенный migrate binary

```powershell
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5433/marketplace?sslmode=disable"
go run ./cmd/migrate up
go run ./cmd/migrate status
go run ./cmd/migrate version
```

### Через Docker image

```powershell
docker compose run --rm api /app/migrate up
```

Миграции лежат в `migrations/`, а `migrations/embed.go` встраивает их в бинарь.

## Тесты

### Backend

```bash
go test ./...
make test
```

CI запускает:

- миграции на тестовой БД;
- `gofmt`-проверку;
- `golangci-lint`;
- `go test ./... -race -coverprofile=coverage.out`;
- валидацию production compose;
- проверку OpenAPI.

### Frontend

```powershell
cd frontend
npm test
```

### QA / smoke / regression

В репозитории есть рабочие QA-сценарии:

- `scripts/qa-run.ps1`
- `scripts/qa_run.py`
- артефакты складываются в `qa-artifacts/`

Обычный локальный сценарий:

```powershell
docker compose up -d postgres
.\scripts\start-qa-api.ps1
.\scripts\qa-run.ps1
```

> QA-скрипты используют локальные seed-данные и seller-аккаунт из миграций. Если сиды не применены, часть кейсов не сработает.

## Линтинг и форматирование

### Backend

```bash
gofmt -w .
golangci-lint run ./...
make fmt
make lint
```

`make ci` последовательно запускает `fmt`, `lint` и `test`, то есть изменяет файлы через `gofmt` перед прогоном остальных шагов.

### Frontend

```powershell
cd frontend
npm run lint
npm run format
npm run format:check
```

## Makefile-команды

| Команда | Что делает |
| --- | --- |
| `make up` | поднимает локальный compose-стек с `--build` |
| `make down` | останавливает compose и удаляет volumes |
| `make build` | собирает `./cmd/api` |
| `make run` | запускает `go run ./cmd/api` |
| `make migrate-up` | применяет миграции через Goose |
| `make migrate-down` | откатывает последнюю миграцию |
| `make seed` | повторно использует `migrate-up` |
| `make test` | `go test ./... -race -coverprofile=coverage.out` |
| `make lint` | запускает `golangci-lint` |
| `make fmt` | форматирует Go-код через `gofmt -w .` |
| `make ci` | `fmt + lint + test` |
| `make openapi-check` | запускает тесты из `apidocs/` |
| `make frontend-install` | `npm install` в `frontend/` |
| `make frontend-dev` | запускает Vite dev server |
| `make prod-config` | валидирует `docker-compose.prod.yml` на `.env.production.example` |
| `make prod-up` | запускает deploy-скрипт с `.env.production` |
| `make prod-down` | останавливает production compose |
| `make prod-migrate` | запускает production-миграции в `ops` profile |
| `make prod-deploy` | деплой с `RUN_MIGRATIONS=true` |
| `make backup` | делает PostgreSQL backup |
| `make restore` | восстанавливает PostgreSQL из dump |
| `make prune-backups` | удаляет старые backup-файлы |

## Frontend scripts

| Команда | Что делает |
| --- | --- |
| `npm run dev` | стартует Vite dev server |
| `npm run build` | компилирует TypeScript и собирает production bundle |
| `npm run lint` | запускает ESLint |
| `npm run preview` | локальный preview production build |
| `npm run test` | запускает Vitest в CI-режиме |
| `npm run test:watch` | запускает Vitest в watch-режиме |
| `npm run format` | форматирует код через Prettier |
| `npm run format:check` | проверяет форматирование без изменений |

## Production и deploy notes

### Что есть

- `docker-compose.prod.yml` для API, PostgreSQL и monitoring-стека.
- `scripts/deploy-prod.sh` для запуска/обновления production-окружения.
- `scripts/backup-postgres.sh`, `scripts/restore-postgres.sh`, `scripts/prune-backups.sh`.
- GitHub Actions:
  - `ci.yml` для проверки кода;
  - `cd.yml` для публикации Docker image и ручного SSH deploy;
  - `release.yml` для tag-based Docker release.

### Как выглядит production-сценарий

1. На сервере должен существовать `.env.production`.
2. `scripts/deploy-prod.sh` поднимает `postgres`, при необходимости прогоняет миграции, затем стартует `api`, `prometheus`, `alertmanager`, `grafana`.
3. В `cd.yml` предусмотрен ручной deploy через `workflow_dispatch` и SSH.

Практические команды:

```bash
make prod-config
make prod-migrate
make prod-up
make backup
```

### Что важно помнить

- Production compose биндует сервисы на `127.0.0.1`, а не на все интерфейсы.
- Frontend production hosting в этом репозитории явно не описан: [нужно уточнить].
- Нейминг публикуемых Docker-образов в `cd.yml` и `release.yml` различается; перед жёсткой автоматизацией это стоит перепроверить.

## Backup / restore

### Backup

```bash
make backup
```

Или напрямую:

```bash
bash scripts/backup-postgres.sh
```

Скрипт:

- читает `.env.production`;
- создаёт `pg_dump -Fc --no-owner --no-privileges`;
- сохраняет файл в `BACKUP_DIR` или в локальный `backups/`.

### Restore

```bash
make restore BACKUP_FILE=/path/to/marketplace_20260101T010203Z.dump
```

Или:

```bash
bash scripts/restore-postgres.sh /path/to/backup.dump
```

Restore-скрипт:

- останавливает `api`;
- выполняет `pg_restore --clean --if-exists`;
- поднимает `api` обратно.

### Очистка старых backup'ов

```bash
make prune-backups
```

Удаляются файлы `marketplace_*.dump` старше `BACKUP_RETENTION_DAYS`.

## Troubleshooting

### `docker compose` не стартует локально

- Убедитесь, что запущен Docker Desktop.
- `scripts/dev-all.ps1` умеет сам попытаться поднять Docker Desktop, если он не доступен.

### Backend не поднимается из-за конфига

- Проверьте `JWT_SECRET`: валидатор требует минимум 32 символа.
- Проверьте `DATABASE_URL` и доступность PostgreSQL на `localhost:5433`.
- Проверьте `APP_ENV`: допустимы только `development`, `test`, `production`.

### Frontend не может авторизоваться

- Сверьте `VITE_AUTH_MODE` и backend-режим `AUTH_COOKIE_MODE`.
- Для cookie-mode браузер должен получать cookies и CSRF token.
- При неправильной связке `cookie/token` клиент будет получать `401` или не сможет обновить сессию.

### Не приходят письма и SMS

- Это ожидаемо для текущего состояния проекта.
- Email/SMS не уходят во внешний провайдер; смотрите `email_jobs`, backend-логи и QA-артефакты.

### Swagger открывается, но API не готов

- Проверяйте `http://localhost:8080/readyz`.
- Для локальной среды посмотрите `docker compose logs -f api`.

### QA-скрипты падают

- Убедитесь, что применены миграции и сиды.
- Проверьте, что backend поднят на `http://localhost:18080`, если используете `start-qa-api.ps1`.
- Проверьте наличие локального контейнера `marketplace-postgres`.

## Roadmap / возможные следующие улучшения

- Подключить реальный SMTP/email provider и SMS provider вместо логирующих транспортов.
- Явно описать production-стратегию для frontend.
- Добавить отдельный CI для frontend build/lint/test, если он нужен в обязательном пайплайне.
- Формализовать release image naming и deploy-поток, чтобы убрать неоднозначность между workflow.
- Добавить отдельный документ по доменной модели и бизнес-правилам checkout/recommendations, если проект будет расти.

## Contributing

Базовый практический процесс:

1. Создайте рабочую ветку.
2. Поднимите локальное окружение через `scripts/dev-all.ps1` или вручную.
3. Перед PR прогоните минимум:
   - `go test ./...`
   - `golangci-lint run ./...`
   - `cd frontend && npm run lint && npm run test`
   - `make openapi-check`
4. Если меняете API-контракт, синхронизируйте код и `apidocs/openapi.yaml`.
5. Не коммитьте реальные секреты, `.env.production` и приватные backup-файлы.

## License

Файл лицензии в текущем репозитории не найден: [нужно уточнить].
