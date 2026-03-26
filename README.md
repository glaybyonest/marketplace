# Marketplace

Go-бэкенд универсального маркетплейса с frontend на React/Vite в том же репозитории.

Сейчас проект покрывает:
- JWT auth с `access/refresh` токенами
- подтверждение email после регистрации
- восстановление пароля по одноразовой ссылке
- универсальный каталог товаров и категории
- изображения товаров, галерея и характеристики (`brand`, `unit`, `specs`)
- улучшенный поиск: suggestions, popular queries, price filters, stock filter
- избранное
- адреса пользователя (`places`)
- серверную корзину
- checkout и заказы
- роль продавца, профиль магазина и витрина продавца
- кабинет продавца, каталог магазина и сводка продавца по заказам
- персональные рекомендации
- background jobs: email queue, cleanup expired auth data, refresh aggregates/recommendations
- security hardening: rate limit на auth endpoints, login lockout/cooldown, security headers, расширенный audit trail
- управление сессиями: session inventory, revoke current/other sessions, logout-all
- optional cookie auth mode с CSRF double-submit защитой

## Требования
- Docker Desktop 4+
- Docker Compose v2
- Node.js 20+ и npm 10+
- Go 1.24+ для локального запуска backend без контейнера API

## Технологии
- Go 1.24+
- PostgreSQL 16
- Chi
- pgx
- Goose migrations
- Docker / Docker Compose
- React + Vite (`frontend/`)

## Структура репозитория
- `cmd/api` - точка входа backend
- `cmd/migrate` - CLI для применения embedded goose-миграций
- `apidocs` - OpenAPI спецификация и Swagger UI handler
- `internal/app` - сборка зависимостей и запуск приложения
- `internal/jobs` - background jobs runner: cleanup, email queue, aggregates, recommendation refresh
- `internal/http` - router, handlers, middleware, DTO, response envelope
- `internal/usecase` - бизнес-логика
- `internal/repository/postgres` - PostgreSQL репозитории
- `internal/domain` - доменные модели и ошибки
- `internal/mailer` - email queue и dev-delivery через логгер
- `migrations` - миграции БД
- `docker` - init-скрипты PostgreSQL
- `frontend` - клиентское приложение
- `scripts` - скрипты локального запуска

## Переменные окружения
Создайте `.env` на основе `.env.example`.

Пример:
```env
APP_ENV=development
HTTP_PORT=8080
DATABASE_URL=postgres://postgres:postgres@postgres:5432/marketplace?sslmode=disable
TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5433/marketplace_test?sslmode=disable
JWT_SECRET=replace_with_a_long_random_secret_of_at_least_32_chars
APP_BASE_URL=http://localhost:5173
MAIL_FROM=no-reply@marketplace.local
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=720h
EMAIL_VERIFY_TTL=24h
PASSWORD_RESET_TTL=1h
LOG_LEVEL=info
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=admin
JOBS_ENABLED=true
JOB_CLEANUP_INTERVAL=1h
JOB_EMAIL_POLL_INTERVAL=5s
JOB_EMAIL_LOCK_TTL=2m
JOB_EMAIL_BATCH_SIZE=20
JOB_EMAIL_MAX_ATTEMPTS=5
JOB_EMAIL_RETENTION=168h
JOB_STATS_REFRESH_INTERVAL=10m
JOB_RECOMMENDATIONS_REFRESH_INTERVAL=15m
JOB_RECOMMENDATION_ACTIVITY_WINDOW=168h
JOB_RECOMMENDATION_USER_BATCH_SIZE=200
JOB_RECOMMENDATION_LIMIT=20
AUTH_LOGIN_FAILURE_WINDOW=15m
AUTH_LOGIN_MAX_ATTEMPTS=5
AUTH_LOGIN_LOCKOUT_DURATION=15m
AUTH_RATE_LIMIT_REGISTER=5
AUTH_RATE_LIMIT_REGISTER_WINDOW=1m
AUTH_RATE_LIMIT_LOGIN=10
AUTH_RATE_LIMIT_LOGIN_WINDOW=1m
AUTH_RATE_LIMIT_REFRESH=30
AUTH_RATE_LIMIT_REFRESH_WINDOW=1m
AUTH_RATE_LIMIT_PASSWORD_RESET=5
AUTH_RATE_LIMIT_PASSWORD_RESET_WINDOW=15m
AUTH_RATE_LIMIT_VERIFY_EMAIL=5
AUTH_RATE_LIMIT_VERIFY_EMAIL_WINDOW=15m
AUTH_COOKIE_MODE=false
AUTH_COOKIE_SECURE=false
AUTH_COOKIE_DOMAIN=
AUTH_COOKIE_SAME_SITE=lax
AUTH_CSRF_ENABLED=true
```

Пояснения:
- `APP_BASE_URL` нужен для ссылок подтверждения email и сброса пароля.
- `MAIL_FROM` используется как адрес отправителя.
- В dev-конфигурации письма сначала попадают в `email_jobs`, а затем background worker пишет их в логи контейнера `api`.
- `GRAFANA_ADMIN_USER` и `GRAFANA_ADMIN_PASSWORD` задают локальный доступ в Grafana.
- `JOBS_ENABLED` включает background jobs.
- `JOB_*` переменные управляют cleanup, email queue, агрегатами и refresh рекомендаций.
- `AUTH_LOGIN_*` переменные управляют cooldown/lockout после неудачных логинов.
- `AUTH_RATE_LIMIT_*` переменные управляют rate limit для чувствительных auth endpoints.
- `AUTH_COOKIE_MODE=true` включает httpOnly cookie auth вместо хранения токенов на frontend.
- `AUTH_COOKIE_SAME_SITE`, `AUTH_COOKIE_SECURE`, `AUTH_COOKIE_DOMAIN` управляют cookie-политикой.
- `AUTH_CSRF_ENABLED=true` включает CSRF double-submit проверку для cookie auth.

Frontend env:
```env
VITE_API_BASE_URL=/api
VITE_API_PROXY_TARGET=http://localhost:8080
VITE_AUTH_MODE=token
```

- `VITE_AUTH_MODE=token` оставляет текущий bearer/localStorage flow.
- `VITE_AUTH_MODE=cookie` переводит frontend на cookie auth с автоматической отправкой `X-CSRF-Token`.

Production env examples:
- root backend env: `.env.production.example`
- frontend production env: `frontend/.env.production.example`

## Быстрый старт
Один скрипт из корня репозитория:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/dev-all.ps1
```

Скрипт:
- создаёт `.env` из `.env.example`, если файла нет
- поднимает `postgres`, `api`, `adminer`, `prometheus`, `grafana`

## Seed-данные и роли

После применения миграций проект поднимается с универсальным каталогом и ролями:
- `customer` — покупательский сценарий: каталог, корзина, checkout, заказы, адреса, избранное
- `seller` — магазин продавца, товары, витрина и seller order summary
- `admin` — backoffice для категорий и товаров

Seller seed-аккаунты для локальной проверки:
- `merchant-urbanwave@seed.marketplace.local` / `Seller123`
- `merchant-casaluna@seed.marketplace.local` / `Seller123`
- `merchant-roamfit@seed.marketplace.local` / `Seller123`

Ключевые миграции для универсализации каталога и seller-flow:
- `migrations/00015_seller_marketplace_universalization.sql`
- `migrations/00016_catalog_refresh.sql`
- `migrations/00017_local_media_refs.sql`

Генератор большого каталога под пользовательский ассортимент:
- `scripts/catalog_refresh_source.txt`
- `scripts/generate_catalog_refresh.mjs`
- локальные media refs для товаров генерируются в формате `marketplace-media://...` и рендерятся на frontend без внешних image-сервисов
- ждёт `http://localhost:8080/readyz`
- ставит frontend-зависимости при необходимости
- запускает `frontend` через `npm run dev`

Опции:
```powershell
# пропустить npm install
powershell -ExecutionPolicy Bypass -File scripts/dev-all.ps1 -SkipFrontendInstall

# остановить docker compose при завершении скрипта
powershell -ExecutionPolicy Bypass -File scripts/dev-all.ps1 -StopContainersOnExit
```

## Запуск вручную
1. Поднимите сервисы:
```bash
docker compose up -d --build
```
2. Проверьте backend:
```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```
3. Запустите frontend:
```bash
cd frontend
npm install
npm run dev
```

Адреса:
- frontend: `http://localhost:5173`
- backend: `http://localhost:8080`
- Swagger UI: `http://localhost:8080/docs/`
- Adminer: `http://localhost:8081`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000`

Остановка:
```bash
docker compose down -v
```

## Миграции
Применить:
```bash
go run github.com/pressly/goose/v3/cmd/goose@v3.26.0 -dir migrations postgres "postgres://postgres:postgres@localhost:5433/marketplace?sslmode=disable" up
```

Откатить один шаг:
```bash
go run github.com/pressly/goose/v3/cmd/goose@v3.26.0 -dir migrations postgres "postgres://postgres:postgres@localhost:5433/marketplace?sslmode=disable" down
```

Ключевые миграции:
- `00003_db_hardening.sql` - ограничения целостности и индексы
- `00004_products_search_trgm.sql` - ускорение `LIKE`-поиска
- `00005_cart_orders.sql` - корзина и заказы
- `00006_auth_email_flows.sql` - подтверждение email и reset tokens
- `00007_product_media_attributes.sql` - изображения товаров, галерея и характеристики
- `00008_search_upgrade.sql` - popular queries, search suggestions и расширенные фильтры каталога
- `00010_observability.sql` - audit logs, error events и observability storage
- `00011_background_jobs.sql` - email_jobs, product_popularity_stats и user_recommendations
- `00012_security_hardening.sql` - login lockout/cooldown поля пользователя
- `00013_session_inventory_cookie_auth.sql` - session inventory (`last_seen_at`) и база для cookie auth

## Локальный запуск backend без API-контейнера
1. Поднимите только PostgreSQL:
```bash
docker compose up -d postgres
```
2. Примените миграции.
3. Запустите API:
```bash
go run ./cmd/api
```

## Проверки
Backend:
```bash
go test ./...
```

Frontend:
```bash
cd frontend
npm run lint
npm run test
npm run build
```

Production config validation:
```bash
ENV_FILE=.env.production.example docker compose --env-file .env.production.example -f docker-compose.prod.yml config
```

## Observability
- `GET /metrics` - Prometheus metrics endpoint
- Prometheus в `docker compose` поднимается на `http://localhost:9090`
- Grafana в `docker compose` поднимается на `http://localhost:3000`
- default login для локальной Grafana берётся из `.env`:
  - `GRAFANA_ADMIN_USER`
  - `GRAFANA_ADMIN_PASSWORD`
- в Grafana автоматически провиженятся:
  - datasource `Prometheus`
  - dashboard `Marketplace Observability`
- JSON structured logs через `slog`
- request-level метрики:
  - `marketplace_http_requests_total`
  - `marketplace_http_request_duration_seconds`
  - `marketplace_http_requests_in_flight`
- database pool metrics:
  - `marketplace_db_pool_total_connections`
  - `marketplace_db_pool_idle_connections`
  - `marketplace_db_pool_acquired_connections`
  - и связанные счетчики acquire/wait/new connections
- error tracking:
  - таблица `error_events`
  - panic и internal error capture с `request_id`, route, status и details
- audit log:
  - таблица `audit_logs`
  - auth/profile/admin события пишутся в аудит
  - admin CRUD покрыт событиями:
    - `admin.categories_listed`
    - `admin.products_listed`
    - `admin.category_created`
    - `admin.category_updated`
    - `admin.category_deleted`
    - `admin.product_created`
    - `admin.product_updated`
    - `admin.product_stock_updated`
    - `admin.product_deleted`
  - security/auth hardening добавляет события:
    - `security.rate_limited`
    - `auth.login_locked`
    - `auth.login_lockout_triggered`
    - `auth.refresh_failed`
    - `auth.logout_failed`
    - `auth.verify_email_failed`
    - `auth.password_reset_failed`

## Security Hardening
- rate limit на:
  - `POST /api/v1/auth/register`
  - `POST /api/v1/auth/login`
  - `POST /api/v1/auth/refresh`
  - `POST /api/v1/auth/verify-email/request`
  - `POST /api/v1/auth/verify-email/confirm`
  - `POST /api/v1/auth/password-reset/request`
  - `POST /api/v1/auth/password-reset/confirm`
- login lockout/cooldown:
  - считаются неудачные попытки логина
  - после достижения порога пользователь временно блокируется
  - успешный логин и password reset сбрасывают счетчики
- security headers:
  - `X-Content-Type-Options: nosniff`
  - `X-Frame-Options: DENY`
  - `Referrer-Policy: no-referrer`
  - `Permissions-Policy`
  - `Cross-Origin-Opener-Policy: same-origin`
  - `Strict-Transport-Security` для HTTPS
- auth routes отдаются с `Cache-Control: no-store`
- session inventory:
  - `GET /api/v1/auth/sessions` показывает активные refresh sessions пользователя
  - `DELETE /api/v1/auth/sessions/{id}` отзывает конкретную сессию
  - `POST /api/v1/auth/logout-all` отзывает все сессии пользователя
- session/device management:
  - access JWT теперь содержит `session_id`
  - защищённые запросы проверяют, что session ещё активна; revoked/rotated session режется сразу, не дожидаясь истечения access token
  - `user_sessions.last_seen_at` обновляется на авторизованных запросах
  - frontend-страница `/account/sessions` показывает device/session inventory
- optional cookie auth mode:
  - access и refresh токены могут жить в `httpOnly` cookies
  - mutating requests в cookie mode требуют `X-CSRF-Token`
  - frontend поддерживает оба режима: `token` и `cookie`

## Background Jobs
- cleanup job:
  - удаляет истёкшие `user_sessions`
  - удаляет `revoked/rotated` session записи после короткого retention-window, чтобы не ломать reuse detection
  - удаляет истёкшие `user_action_tokens`
  - удаляет использованные `user_action_tokens` после короткого retention-window
  - переоткрывает зависшие `email_jobs`
  - очищает старые `sent/failed` email jobs
- email job:
  - auth-flow теперь ставит email в очередь `email_jobs`
  - worker забирает pending jobs и доставляет их через текущий mailer
  - в dev-режиме фактическая доставка остаётся логируемой в `api`
- aggregates job:
  - пересчитывает `product_popularity_stats` из `favorites` и `user_product_events`
- recommendation job:
  - выбирает активных пользователей по recent activity
  - обновляет `user_recommendations`
  - endpoint `/api/v1/recommendations` сначала использует кэш, затем fallback на online build

## OpenAPI / Swagger
- Swagger UI: `http://localhost:8080/docs/`
- Raw OpenAPI spec: `http://localhost:8080/docs/openapi.yaml`
- Спецификация хранится в `apidocs/openapi.yaml`
- Валидация спецификации входит в `go test ./...`

Проверить только документацию:
```bash
make openapi-check
```

Без `make`:
```bash
go test ./apidocs
```

Что есть в документации:
- все публичные и защищённые endpoints
- bearer auth схема для JWT access token
- единый envelope-формат ответов и ошибок
- примеры request/response payload
- схемы для catalog/auth/cart/orders/admin API

## Production Deploy
Базовый production baseline лежит в:
- `docker-compose.prod.yml`
- `.env.production.example`
- `scripts/deploy-prod.sh`
- `scripts/backup-postgres.sh`
- `scripts/restore-postgres.sh`
- `scripts/prune-backups.sh`
- `.github/workflows/cd.yml`

Что делает production compose:
- поднимает `postgres`, `api`, `prometheus`, `alertmanager`, `grafana`
- использует готовый image через `API_IMAGE`
- не запускает миграции автоматически в `api`, для этого есть отдельный сервис `migrate`
- биндует сервисы на `127.0.0.1`, предполагая внешний reverse proxy / TLS termination

Быстрый production deploy на сервере:
1. Скопируйте `.env.production.example` в `.env.production` и заполните секреты.
2. Настройте `alertmanager.yml`, если нужен реальный webhook/email receiver.
3. Выполните:
```bash
ENV_FILE=.env.production bash scripts/deploy-prod.sh
```
4. Проверьте:
```bash
curl http://127.0.0.1:8080/readyz
curl http://127.0.0.1:9090/-/ready
curl http://127.0.0.1:9093/-/ready
curl http://127.0.0.1:3000/api/health
```

Frontend production build:
```bash
cd frontend
cp .env.production.example .env.production
npm ci
npm run build
```

Manual migration commands:
```bash
docker compose --env-file .env.production -f docker-compose.prod.yml --profile ops run --rm migrate up
docker compose --env-file .env.production -f docker-compose.prod.yml --profile ops run --rm migrate version
```

## CI/CD
Workflow:
- `ci.yml` — форматирование, lint, тесты, валидация production compose
- `cd.yml` — build/push Docker image в GHCR и manual deploy на production host

`cd.yml` публикует image:
- `ghcr.io/<owner>/marketplace-backend-api:latest`
- `ghcr.io/<owner>/marketplace-backend-api:<git-sha>`

Для deploy workflow нужны GitHub Secrets:
- `DEPLOY_HOST`
- `DEPLOY_PORT`
- `DEPLOY_USER`
- `DEPLOY_SSH_KEY`
- `DEPLOY_PATH`
- `GHCR_USERNAME`
- `GHCR_TOKEN`

Ожидания от сервера:
- установлен Docker + Docker Compose v2
- в `DEPLOY_PATH` уже лежит `.env.production`
- внешний reverse proxy публикует API/Grafana/Alertmanager при необходимости

## Backup / Restore
Создать backup:
```bash
ENV_FILE=.env.production bash scripts/backup-postgres.sh
```

Восстановить из backup:
```bash
ENV_FILE=.env.production bash scripts/restore-postgres.sh /path/to/marketplace_YYYYMMDDTHHMMSSZ.dump
```

Очистить старые backup-файлы:
```bash
ENV_FILE=.env.production bash scripts/prune-backups.sh
```

Рекомендуемая стратегия:
- nightly `pg_dump -Fc`
- хранить минимум `14` дней локально
- после создания backup копировать файл во внешнее хранилище (`S3`, `Backblaze`, `rsync` target)
- минимум раз в месяц делать test restore в отдельную БД/стенд

Пример cron:
```cron
0 2 * * * cd /opt/marketplace && ENV_FILE=.env.production bash scripts/backup-postgres.sh && ENV_FILE=.env.production bash scripts/prune-backups.sh
```

## Monitoring / Alerts
Production monitoring config:
- `docker/monitoring/prometheus/prometheus.prod.yml`
- `docker/monitoring/prometheus/alerts.prod.yml`
- `docker/monitoring/alertmanager/alertmanager.yml`
- `docker/monitoring/alertmanager/alertmanager.example.yml`

В baseline уже есть alerts:
- `MarketplaceAPIDown`
- `MarketplaceHigh5xxRate`
- `MarketplaceHighLatencyP95`
- `MarketplaceDBPoolSaturation`

По умолчанию `alertmanager.yml` использует `null` receiver, чтобы production compose стартовал безопасно.
Для реальных уведомлений:
1. замените receiver в `docker/monitoring/alertmanager/alertmanager.yml`
2. либо начните с шаблона `docker/monitoring/alertmanager/alertmanager.example.yml`

## API
Базовый адрес: `http://localhost:8080`

Публичные endpoints:
- `GET /healthz`
- `GET /readyz`
- `GET /metrics`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/verify-email/request`
- `POST /api/v1/auth/verify-email/confirm`
- `POST /api/v1/auth/password-reset/request`
- `POST /api/v1/auth/password-reset/confirm`
- `GET /api/v1/categories`
- `GET /api/v1/categories/slug/{slug}`
- `GET /api/v1/categories/{id}`
- `GET /api/v1/search/suggestions`
- `GET /api/v1/search/popular`
- `GET /api/v1/products`
- `GET /api/v1/products/slug/{slug}`
- `GET /api/v1/products/{id}`

Требуют авторизацию:
- `POST /api/v1/auth/logout`
- `POST /api/v1/auth/logout-all`
- `GET /api/v1/auth/me`
- `GET /api/v1/auth/sessions`
- `DELETE /api/v1/auth/sessions/{id}`
- `GET /api/v1/profile`
- `PATCH /api/v1/profile`
- `GET /api/v1/favorites`
- `POST /api/v1/favorites/{product_id}`
- `DELETE /api/v1/favorites/{product_id}`
- `GET /api/v1/cart`
- `POST /api/v1/cart/items`
- `PATCH /api/v1/cart/items/{product_id}`
- `DELETE /api/v1/cart/items/{product_id}`
- `DELETE /api/v1/cart`
- `POST /api/v1/places`
- `GET /api/v1/places`
- `PATCH /api/v1/places/{id}`
- `DELETE /api/v1/places/{id}`
- `POST /api/v1/orders`
- `GET /api/v1/orders`
- `GET /api/v1/orders/{id}`
- `GET /api/v1/recommendations`

## Сквозные сценарии
Регистрация и заказ:
1. Зарегистрируйтесь.
2. Откройте ссылку подтверждения email из логов `api`.
3. Войдите в аккаунт.
4. Добавьте товары в корзину.
5. Создайте адрес в `My places`.
6. Откройте `Checkout` и оформите заказ.
7. Проверьте заказ в `Orders`.

Восстановление пароля:
1. Откройте `/forgot-password`.
2. Запросите ссылку восстановления.
3. Откройте ссылку из логов `api`.
4. Установите новый пароль на `/reset-password`.
5. Войдите с новым паролем.

Поиск и каталог:
1. Откройте главную страницу.
2. Начните вводить запрос в строке поиска и выберите suggestion или popular query.
3. Дополнительно ограничьте выдачу по категории, цене и наличию.
4. Откройте карточку товара из результатов.

Seller flow:
1. Войдите под seller seed-аккаунтом или откройте `/seller` из обычного аккаунта.
2. Если аккаунт ещё не seller, заполните профиль магазина и активируйте кабинет продавца.
3. Перейдите в `/seller/products` и добавьте или обновите карточку товара.
4. Проверьте `/seller/storefront` для описания, контактов и статуса магазина.
5. Откройте `/seller/orders`, чтобы увидеть seller-side сводку по заказам.

## Частые проблемы
- `404` на `api/api/v1/...`: `VITE_API_BASE_URL` должен быть `/api`, а frontend должен ходить на `/v1/...`
- `400` на `POST /api/v1/auth/register`: пароль должен быть длиной `8-72`, содержать минимум одну латинскую букву и одну цифру; `email` должен быть уникальным
- `400` на `POST /api/v1/auth/verify-email/confirm` или `POST /api/v1/auth/password-reset/confirm`: ссылка истекла, уже использована или повреждена; запросите новую
- `404` или `500` на корзине/заказах после обновления схемы: примените `goose up`, чтобы в БД были миграции `00005_cart_orders.sql` и `00006_auth_email_flows.sql`
- не работают suggestions/popular search или новые price filters: убедитесь, что применена миграция `00008_search_upgrade.sql`
- письмо "не пришло": для локальной разработки письмо пишется в `docker compose logs -f api`
- `css2 ... ERR_TIMED_OUT`: проблема сети/кэша браузера, сделайте hard reload

## Каталог и поиск
`GET /api/v1/products` поддерживает query params:
- `q`
- `category_id`
- `min_price`
- `max_price`
- `in_stock`
- `sort`
- `page`
- `limit`

Search endpoints:
- `GET /api/v1/search/suggestions?q=headphones&limit=8`
- `GET /api/v1/search/popular?limit=6`

## Полезные команды
- `powershell -ExecutionPolicy Bypass -File scripts/dev-all.ps1`
- `powershell -ExecutionPolicy Bypass -File scripts/dev-backend.ps1`
- `powershell -ExecutionPolicy Bypass -File scripts/dev-frontend.ps1`
- `docker compose logs -f api`
- `go test ./...`

## Лицензия
Внутренний / учебный проект.

## Admin Backoffice

Для доступа к административному интерфейсу укажите email администратора в `.env`:

```env
ADMIN_EMAILS=admin@example.com
```

Как это работает:
- при старте backend все пользователи из `ADMIN_EMAILS` получают роль `admin`
- если новый пользователь регистрируется с email из `ADMIN_EMAILS`, он тоже создаётся как `admin`
- после входа администратору становятся доступны маршруты frontend:
  - `/admin`
  - `/admin/categories`
  - `/admin/products`

Административные endpoints:
- `GET /api/v1/admin/categories`
- `POST /api/v1/admin/categories`
- `PATCH /api/v1/admin/categories/{id}`
- `DELETE /api/v1/admin/categories/{id}`
- `GET /api/v1/admin/products`
- `POST /api/v1/admin/products`
- `PATCH /api/v1/admin/products/{id}`
- `PATCH /api/v1/admin/products/{id}/stock`
- `DELETE /api/v1/admin/products/{id}`

Что поддерживает админка:
- CRUD категорий
- CRUD товаров
- изменение остатков
- скрытие товара из публичного каталога через `is_active`

## Seller API

Маршруты seller-кабинета:
- `/seller`
- `/seller/products`
- `/seller/storefront`
- `/seller/orders`

Seller endpoints:
- `GET /api/v1/seller/profile`
- `PUT /api/v1/seller/profile`
- `PATCH /api/v1/seller/profile`
- `GET /api/v1/seller/dashboard`
- `GET /api/v1/seller/products`
- `POST /api/v1/seller/products`
- `PATCH /api/v1/seller/products/{id}`
- `PATCH /api/v1/seller/products/{id}/stock`
- `DELETE /api/v1/seller/products/{id}`
- `GET /api/v1/seller/orders`

Для применения роли и admin API нужна миграция:
- `00009_admin_backoffice.sql`
