# Marketplace Backend

Go-бэкенд маркетплейса с frontend на React/Vite в том же репозитории.

Сейчас проект покрывает:
- JWT auth с `access/refresh` токенами
- подтверждение email после регистрации
- восстановление пароля по одноразовой ссылке
- каталог товаров и категории
- изображения товаров, галерея и характеристики (`brand`, `unit`, `specs`)
- улучшенный поиск: suggestions, popular queries, price filters, stock filter
- избранное
- адреса пользователя (`places`)
- серверную корзину
- checkout и заказы
- персональные рекомендации

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
- `internal/app` - сборка зависимостей и запуск приложения
- `internal/http` - router, handlers, middleware, DTO, response envelope
- `internal/usecase` - бизнес-логика
- `internal/repository/postgres` - PostgreSQL репозитории
- `internal/domain` - доменные модели и ошибки
- `internal/mailer` - отправка email через логгер в dev-режиме
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
```

Пояснения:
- `APP_BASE_URL` нужен для ссылок подтверждения email и сброса пароля.
- `MAIL_FROM` используется как адрес отправителя.
- В dev-конфигурации письма не уходят во внешний SMTP: backend пишет их в логи контейнера `api`.

Frontend env:
```env
VITE_API_BASE_URL=/api
VITE_API_PROXY_TARGET=http://localhost:8080
```

## Быстрый старт
Один скрипт из корня репозитория:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/dev-all.ps1
```

Скрипт:
- создаёт `.env` из `.env.example`, если файла нет
- поднимает `postgres`, `api`, `adminer`
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
- Adminer: `http://localhost:8081`

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

## Observability
- `GET /metrics` - Prometheus metrics endpoint
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
  - auth/profile события пишутся в аудит (`register/login/logout/refresh/email verify/password reset/profile update`)

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
- `GET /api/v1/auth/me`
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
- `GET /api/v1/search/suggestions?q=cement&limit=8`
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

Для применения роли и admin API нужна миграция:
- `00009_admin_backoffice.sql`
