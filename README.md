# Marketplace Backend

Бэкенд маркетплейса на Go с JWT-аутентификацией, каталогом, избранным, адресами и персональными рекомендациями.

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

## Запуск через Docker
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

## Локальный запуск frontend
```bash
cd frontend
npm install
npm run dev
```

По умолчанию фронтенд проксирует запросы на `http://localhost:8080`.

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
- `POST /api/v1/places`
- `GET /api/v1/places`
- `PATCH /api/v1/places/{id}`
- `DELETE /api/v1/places/{id}`
- `GET /api/v1/recommendations`

## Полезные команды
Из корня репозитория:
- `powershell -ExecutionPolicy Bypass -File scripts/dev-backend.ps1`
- `powershell -ExecutionPolicy Bypass -File scripts/dev-frontend.ps1`
- `go test ./...`

## Лицензия
Внутренний / учебный проект.
