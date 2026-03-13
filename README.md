# Marketplace Backend

Бэкенд маркетплейса на Go с JWT-аутентификацией, каталогом, избранным, корзиной, checkout, заказами, адресами и персональными рекомендациями.

## Требования
- Docker Desktop 4+
- Docker Compose v2
- Node.js 20+ и npm 10+ (для локального frontend)
- Go 1.24+ (для локального backend без API-контейнера)

## Технологии
- Go 1.24+
- PostgreSQL 16
- Chi router
- Pgx
- Goose migrations
- Docker / Docker Compose
- Frontend: React + Vite (в папке `frontend/`)

## Структура репозитория
- `cmd/api` - точка входа приложения
- `internal/app` - инициализация и сборка зависимостей
- `internal/http` - handlers, middleware, DTO, envelope-ответы
- `internal/usecase` - бизнес-логика
- `internal/repository/postgres` - SQL-репозитории
- `internal/domain` - доменные модели и ошибки
- `migrations` - миграции схемы и сиды
- `docker` - init-скрипты PostgreSQL
- `frontend` - фронтенд, интегрированный с этим API
- `scripts` - вспомогательные скрипты запуска

## Что уже реализовано
- JWT auth с access/refresh token и logout/revoke
- каталог товаров и категории
- избранное
- адреса пользователя (`places`)
- серверная корзина
- checkout из корзины по сохраненному адресу
- история заказов
- персональные рекомендации

## Переменные окружения
Создайте `.env` на основе `.env.example` и задайте безопасный `JWT_SECRET`.

Пример:
```env
APP_ENV=development
HTTP_PORT=8080
DATABASE_URL=postgres://postgres:postgres@postgres:5432/marketplace?sslmode=disable
TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5433/marketplace_test?sslmode=disable
JWT_SECRET=replace_with_a_long_random_secret_of_at_least_32_chars
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=720h
LOG_LEVEL=info
```

## Быстрый старт (одним скриптом)
Из корня репозитория:
```powershell
powershell -ExecutionPolicy Bypass -File scripts/dev-all.ps1
```

Что делает скрипт:
- создаёт `.env` из `.env.example`, если файла нет;
- поднимает `postgres`, `api`, `adminer` через `docker compose`;
- ждёт готовность backend (`/readyz`);
- запускает frontend (`npm run dev`).

Опции:
```powershell
# пропустить npm install во frontend
powershell -ExecutionPolicy Bypass -File scripts/dev-all.ps1 -SkipFrontendInstall

# при выходе из скрипта автоматически остановить docker-контейнеры
powershell -ExecutionPolicy Bypass -File scripts/dev-all.ps1 -StopContainersOnExit
```

## Запуск вручную через Docker
1. Поднять сервисы:
```bash
docker compose up -d --build
```
2. Проверить статус:
```bash
docker compose ps
```
3. Проверить health endpoints:
```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```
4. Остановить сервисы:
```bash
docker compose down -v
```

## Локальный запуск frontend
```bash
cd frontend
npm install
npm run dev
```

По умолчанию фронтенд проксирует запросы на `http://localhost:8080`.

`frontend/.env.example`:
```env
VITE_API_BASE_URL=/api
VITE_API_PROXY_TARGET=http://localhost:8080
```

## Миграции базы данных
Применить миграции с хоста:
```bash
go run github.com/pressly/goose/v3/cmd/goose@v3.26.0 -dir migrations postgres "postgres://postgres:postgres@localhost:5433/marketplace?sslmode=disable" up
```

Откатить один шаг:
```bash
go run github.com/pressly/goose/v3/cmd/goose@v3.26.0 -dir migrations postgres "postgres://postgres:postgres@localhost:5433/marketplace?sslmode=disable" down
```

Примечания:
- `00003_db_hardening.sql` добавляет ограничения целостности и рабочие индексы.
- `00004_products_search_trgm.sql` добавляет опциональный `pg_trgm` индекс для ускорения `LIKE`-поиска.
- `00005_cart_orders.sql` добавляет таблицы корзины и заказов: `cart_items`, `orders`, `order_items`.

## Локальный запуск backend (без Docker API-контейнера)
1. Поднять только PostgreSQL:
```bash
docker compose up -d postgres
```
2. Применить миграции.
3. Запустить API:
```bash
go run ./cmd/api
```

## Тесты и проверки
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

## Обзор API
Базовый адрес: `http://localhost:8080`

Публичные endpoints:
- `GET /healthz`
- `GET /readyz`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `GET /api/v1/categories`
- `GET /api/v1/categories/slug/{slug}`
- `GET /api/v1/categories/{id}`
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

## Сквозной пользовательский сценарий
1. Зарегистрироваться или войти.
2. Добавить один или несколько товаров в корзину.
3. Создать адрес в `My places`.
4. Открыть `Checkout`, выбрать адрес и оформить заказ.
5. Проверить историю в `Orders`.

## Частые проблемы
- `404` на `api/api/v1/...`: проверьте, что `VITE_API_BASE_URL=/api`, а запросы в коде идут на `/v1/...`.
- `400` на `POST /api/v1/auth/register`: пароль должен быть длиной `8-72`, содержать минимум одну латинскую букву и одну цифру; `email` должен быть уникальным.
- `404` или `500` на корзине/заказах после обновления кода: примените миграции `goose up`, чтобы в БД появились таблицы из `00005_cart_orders.sql`.
- `css2 ... ERR_TIMED_OUT`: проблема с внешними шрифтами/сетью/кэшем браузера; сделайте hard refresh (`Ctrl+F5`) и перезапустите frontend.

## Полезные команды
Из корня репозитория:
- `powershell -ExecutionPolicy Bypass -File scripts/dev-all.ps1`
- `powershell -ExecutionPolicy Bypass -File scripts/dev-backend.ps1`
- `powershell -ExecutionPolicy Bypass -File scripts/dev-frontend.ps1`
- `go test ./...`

## Лицензия
Внутренний / учебный проект.
