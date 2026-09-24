# bookly-backend

Go API для Bookly: запись клиентов к мастерам (маникюр, барбер и т.д.) с поиском по карте, расписанием и входом по телефону.

## Стек

- Go 1.26, Fiber v2
- PostgreSQL 17 + PostGIS + btree_gist, sqlx + pgx
- Redis: rate limit для OTP, кеш
- goose: миграции

## Структура

```text
cmd/
  api/          точка входа HTTP API
  migrate/      запуск миграций goose
  command/      CLI для служебных задач (импорт внешних каталогов)
internal/
  bootstrap/    config.go, deps.go, app.go, routes.go - сборка приложения
  domain/       enum, state machine записи, валидация, ошибки
  usecase/      бизнес-логика по модулям: auth, user, master, catalog, schedule, booking, notification, assistant
  infrastructure/
    postgres/   репозитории (по модулю на папку)
    redis/      клиент Redis
    otp/        провайдеры отправки кода: sms, whatsapp, telegram
    storage/    S3 (Garage) для фото
    sources/    адаптеры внешних каталогов (zapis/)
  transport/rest/v1/  router, handlers, middleware, response
  transport/command/  CLI: разбор аргументов, вызов usecase
  pkg/ctxuser/  текущий пользователь в context
migrations/     SQL миграции goose
```

## Схема БД

| Миграция | Таблицы |
|---|---|
| 00001_extensions | pgcrypto, postgis, btree_gist, функция `set_updated_at()` |
| 00002_create_users_rbac | `users`, `roles`, `user_roles` |
| 00003_create_auth | `auth_identities`, `otp_challenges`, `sessions` |
| 00004_create_profiles | `client_profiles`, `master_profiles` |
| 00005_create_master_locations | `master_locations` (geography Point + GiST) |
| 00006_create_services | `service_categories`, `master_services` |
| 00007_create_schedule | `working_hours`, `schedule_exceptions` |
| 00008_create_bookings | `bookings`, `booking_status_history` |
| 00009_create_notifications | `notifications` |
| 00010_create_assistant_requests | `assistant_requests` |
| 00011_create_import_sources | `data_sources`, `import_runs`, `source_*` (сырые данные внешних каталогов) |

Ключевые решения:

- Enum (статусы, каналы, типы) и бизнес-правила живут в `internal/domain`, а не в CHECK constraints. Новое значение = правка кода, без миграции. Usecase обязан валидировать вход через `domain.Parse*` / `Validate()`. В БД остаются только структурные инварианты: FK, `price >= 0`, `start < end`, защита от пересечений.

- RBAC: `roles` + `user_roles`, один пользователь может быть и клиентом, и мастером.
- Вход по телефону: OTP через SMS, WhatsApp или Telegram Gateway. Храним только hash кода. Пользователь создаётся после успешной проверки.
- Сессии: в `sessions` лежит только hash refresh token, access token - короткий JWT.
- Двойная запись невозможна на уровне БД: exclusion constraint `bookings_no_master_overlap` по `(master_id, booking_period)` для строк с `cancelled_at IS NULL`. При отмене usecase ставит `cancelled_at` (см. `BookingStatus.IsCancelled`).
- Свободные слоты не храним, считаем из `working_hours` - `schedule_exceptions` - `bookings`.
- Цена в minor units (`bigint`), в `bookings` сохраняется snapshot цены и названия услуги.

## Авторизация

Вход только по телефону, без пароля. Все маршруты под `/api/v1/auth`.

| Метод | Путь | Что делает |
|---|---|---|
| POST | `/otp/request` | `{phone, channel}`: отправить код (sms / whatsapp / telegram) |
| POST | `/register` | `{phone, name, code}`: создать пользователя (роль client), вернуть токены |
| POST | `/login` | `{phone, code}`: вернуть токены |
| POST | `/refresh` | `{refresh_token}`: ротация, старый refresh отзывается |
| POST | `/logout` | `{refresh_token}`: отозвать текущую сессию |
| POST | `/logout-all` | Bearer: отозвать все сессии |
| GET | `/me` | Bearer: текущий пользователь |

- Access token: JWT HS256, живёт `ACCESS_TOKEN_TTL` (15m), в claims `sub`, `sid`, `roles`.
- Refresh token: случайные 32 байта, в `sessions` лежит только SHA-256 hash. При повторном использовании уже отозванного refresh token отзываются все сессии пользователя.
- OTP сейчас заглушка: код всегда `OTP_STATIC_CODE`, отправитель только пишет в лог. Защиты уже работают: cooldown повторной отправки, лимит запросов в час на номер (Redis), 5 попыток на код, одноразовость кода.
- Для реальной отправки замените `otp.StaticGenerator` и `otp.LogSender` в `bootstrap/deps.go`.

## API документация

Scalar UI: `http://localhost:8080/docs`, спецификация: `/docs/openapi.json`.

`docs/` не хранится в git: каждый генерирует спецификацию сам, после clone и после изменения handler-ов или DTO:

```bash
make swagger   # swag fmt + swag init -> docs/swagger.json
```

Файл читается на каждый запрос, перезапуск не нужен. Без файла `/docs/openapi.json` отвечает 404 с подсказкой. При сборке Docker-образа `make swagger` нужно запускать в Dockerfile.

Доступ:

- dev: открыто. Если задать `DOCS_USER` / `DOCS_PASSWORD`, включается basic auth.
- prod: basic auth обязателен. Без `DOCS_USER` и `DOCS_PASSWORD` (12+ символов) приложение не стартует. Можно выключить docs целиком: `DOCS_ENABLED=false`.

## Импорт внешних каталогов (CLI)

Стягивает фирмы, услуги и мастеров из публичных каталогов (сейчас zapis.kz) в таблицы `source_*`. Это данные для исследования рынка, они не связаны с `bookings` и не являются источником правды Bookly.

```bash
go run ./cmd/command sources                                  # зарегистрированные источники
go run ./cmd/command cities -source zapis_kz                  # города источника (id для -city)
go run ./cmd/command import -source zapis_kz -city 1 -max-firms 10
go run ./cmd/command import -source zapis_kz -city 1,2 -snapshots
go run ./cmd/command runs                                     # история запусков
```

То же через make: `make import city=1 max=10`.

- Повторный запуск обновляет строки (upsert по `external_id`), `last_seen_at` показывает, когда запись видели последний раз.
- Ошибка одной фирмы не останавливает импорт: она считается в `errors_count`, первые 20 сообщений лежат в `error_summary`.
- Ctrl+C корректно завершает запуск со статусом `cancelled`.
- `-snapshots` сохраняет сырые JSON-ответы в `source_snapshots`, чтобы можно было переразобрать данные без повторных запросов. Полный Алматы занимает около 6 MB.
- Между запросами пауза `IMPORT_REQUEST_DELAY` (500ms). Полный Алматы, около 750 фирм, идёт около 15–20 минут.

Новый источник: реализовать `domain.CatalogSource` в `internal/infrastructure/sources/<name>/` и зарегистрировать в `bootstrap/command.go`.

Соблюдайте условия использования источника: только публичные данные, умеренная частота запросов, без авторизованных и персональных данных клиентов.

## Запуск

Нужен Postgres с PostGIS (образ `postgis/postgis:17-3.5-alpine`) и Redis.

```bash
cp .env.example .env
make migration-up
make run
```

Проверка: `GET /healthz` (процесс жив), `GET /readyz` (Postgres и Redis доступны).

Новая миграция: `make migration-create name=add_reviews`.
