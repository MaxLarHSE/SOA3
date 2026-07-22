# SOA3 — сервис поиска рейсов и бронирования мест

Учебный проект из двух микросервисов: `flight-service` хранит рейсы и резервации мест, `booking-service` предоставляет REST API для бронирований. Бронирование мест — распределённая операция: booking-service резервирует места в flight-service по gRPC и сохраняет бронь в своей БД.

## Архитектура

- **flight-service** — gRPC-сервер (`:50051`). Хранит рейсы (`flights`) и резервации мест (`seat_reservations`) в своей PostgreSQL. Методы: `SearchFlights`, `GetFlight`, `ReserveSeats`, `ReleaseReservation`.
- **booking-service** — HTTP-сервер (`:8080`, роутер chi). Хранит бронирования (`bookings`) в отдельной PostgreSQL. Для операций с местами вызывает flight-service по gRPC через свой клиент (`booking-service/client`), подписывая каждый вызов API-ключом.
- У каждого сервиса **своя база** — сервисы не лезут в чужие таблицы, общаются только через API. Миграции накатывает Flyway при старте compose.

```
            REST (JSON)                 gRPC (+ x-api-key)
  клиент ─────────────► booking-service ─────────────► flight-service
                             │                              │
                             ▼                              ▼
                       booking-db (PG)                flight-db (PG)
                        bookings                    flights, seat_reservations
```

## Технологии

- Go 1.26, модуль `SOA3`
- gRPC (`google.golang.org/grpc`) + protobuf, контракт в [proto/flight_service.proto](proto/flight_service.proto), сгенерированный код в `gen/proto`
- HTTP-роутер `go-chi/chi`, `google/uuid`
- PostgreSQL 15 (`lib/pq`, чистый `database/sql` без ORM), миграции — Flyway
- Docker Compose: 2 БД + 2 flyway-контейнера + 2 сервиса, с healthcheck'ами и порядком запуска

## Ключевые инженерные решения

### 1. `SELECT ... FOR UPDATE` против гонки при резервировании мест

Резервирование в [flight-service/repository.go:246](flight-service/repository.go#L246) выполняется в транзакции:

```sql
BEGIN;
SELECT available_seats FROM flights WHERE id = $1 FOR UPDATE;  -- блокировка строки
-- проверка available_seats >= seat_count в коде
UPDATE flights SET available_seats = available_seats - $1 WHERE id = $2;
INSERT INTO seat_reservations (...) VALUES (..., 'ACTIVE');
COMMIT;
```

Без `FOR UPDATE` два конкурентных запроса могли бы одновременно прочитать `available_seats = 1`, оба пройти проверку и оба списать место — овербукинг. `FOR UPDATE` блокирует строку рейса до конца транзакции: второй запрос ждёт коммита первого и читает уже обновлённое значение. Дополнительная страховка на уровне БД — констрейнт `CHECK (available_seats >= 0)` в [миграции](flight-service/migrations/V1__create_flights.sql). Освобождение мест (`ReleaseReservation`) симметрично блокирует строку резервации `FOR UPDATE`, возвращает места и переводит резервацию в `RELEASED` — повторный вызов не сможет вернуть места дважды, т.к. фильтр `status = 'ACTIVE'`.

### 2. Компенсация при частичном сбое (saga вручную)

`CreateBooking` в [booking-service/http/handler.go:79](booking-service/http/handler.go#L79) делает два шага в разных системах: сначала `ReserveSeats` по gRPC, затем `INSERT` брони в свою БД. Если вставка брони упала — вызывается компенсирующий `ReleaseReservation`, чтобы не оставить «висящие» занятые места. Связка идёт по `booking_id` (UUID генерируется на стороне booking-service до резервирования), в `seat_reservations` он `UNIQUE` — повторная резервация с тем же booking_id невозможна.

### 3. Аутентификация между сервисами по API-ключу

flight-service защищён unary-интерцептором ([flight-service/auth.go:13](flight-service/auth.go#L13)): каждый gRPC-запрос должен нести метаданные `x-api-key`, совпадающие с `FLIGHT_SERVICE_API_KEY`. Клиент в booking-service автоматически подписывает все вызовы через `withAuth` ([booking-service/client/client.go:40](booking-service/client/client.go#L40)). Проверка вынесена в интерцептор, а не в каждый хендлер — методы сервера не знают об аутентификации.

### 4. Разделение слоёв и маппинг ошибок

- **Транспорт → логика → хранение**: HTTP-хендлеры / gRPC-сервер отдельно от репозиториев (`repository.go`), у booking-service отдельные DTO ([http/dto.go](booking-service/http/dto.go)) вместо протаскивания доменных структур наружу.
- gRPC-сервер переводит ошибки хранилища в коды статусов: `sql.ErrNoRows` → `NotFound`, «not enough seats» → `ResourceExhausted`, «reservation is not active» → `FailedPrecondition` ([flight-service/server.go](flight-service/server.go)). Клиент различает бизнес-ошибки без парсинга текста.
- Инварианты продублированы в схеме БД: `available_seats <= total_seats`, `origin <> destination`, `arrival_time > departure_time`, уникальность `(flight_number, departure_time)`.

## Запуск

Нужны Docker и Docker Compose:

```bash
docker compose up --build
```

Compose сам поднимает базы, ждёт healthcheck, прогоняет Flyway-миграции и стартует сервисы.

**Порты:**

| Что | Порт |
|---|---|
| booking-service (REST) | `localhost:8080` |
| flight-service (gRPC) | `localhost:50051` |
| flight-db (PostgreSQL) | `localhost:5433` |
| booking-db (PostgreSQL) | `localhost:5434` |

**Переменные окружения** (заданы в [docker-compose.yml](docker-compose.yml)):

| Переменная | Сервис | Назначение |
|---|---|---|
| `DATABASE_URL` | оба | строка подключения к своей PostgreSQL |
| `GRPC_PORT` | flight-service | порт gRPC (в коде слушается `:50051`) |
| `HTTP_PORT` | booking-service | порт HTTP (по умолчанию `8080`) |
| `FLIGHT_SERVICE_ADDR` | booking-service | адрес flight-service (по умолчанию `localhost:50051`) |
| `FLIGHT_SERVICE_API_KEY` | оба | общий API-ключ для меж-сервисной аутентификации |

## Тесты

Юнит-тесты лежат рядом с кодом, БД и сеть не поднимаются.

**Репозиторий** ([repository_test.go](flight-service/repository_test.go)) — SQL мокается через `DATA-DOG/go-sqlmock`, который проверяет и порядок вызовов внутри транзакции (`BeginTx → SELECT ... FOR UPDATE → UPDATE/INSERT → Commit/Rollback`):

- `ReserveSeats` — успешная бронь; нехватка мест (возвращается `ErrNotEnoughSeats`, транзакция **откатывается**, а не коммитится); повторная бронь с тем же `booking_id` (нарушение `UNIQUE` → откат); рейс не найден.
- `ReleaseReservation` — успешный возврат мест; релиз неактивной брони (`sql.ErrNoRows`, откат).
- `mapFlightStatus` — все ветки маппинга статусов, включая неизвестный статус → `UNSPECIFIED`.

**gRPC-слой** ([server_test.go](flight-service/server_test.go)) — сервер зависит от интерфейса `FlightRepository`, в тестах подставляется фейковый репозиторий; проверяется маппинг доменных ошибок в gRPC-коды:

- `ReserveSeats`: `ErrNotEnoughSeats` → `ResourceExhausted`, `sql.ErrNoRows` → `NotFound`, успех → `Success: true`.
- `ReleaseReservation`: `ErrReservationNotActive` → `FailedPrecondition`, `sql.ErrNoRows` → `NotFound`.
- `GetFlight`: `sql.ErrNoRows` → `NotFound`, прочие ошибки → `Internal`.

Запуск:

```bash
go test ./...
# подробно:
go test -v ./flight-service/
```

## Примеры запросов

REST (см. также `test.http`):

```bash
# поиск рейсов
curl "http://localhost:8080/flights?origin=SVO&destination=LED&date=2026-07-22"

# создать бронирование
curl -X POST http://localhost:8080/bookings \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "11111111-1111-1111-1111-111111111111",
    "flight_id": "<flight-uuid>",
    "passenger_name": "Ivan Ivanov",
    "passenger_email": "ivan@example.com",
    "seat_count": 2
  }'

# бронирования пользователя / бронь по id / отмена
curl "http://localhost:8080/bookings?user_id=11111111-1111-1111-1111-111111111111"
curl "http://localhost:8080/bookings/<booking-uuid>"
curl -X POST "http://localhost:8080/bookings/<booking-uuid>/cancel"
```

gRPC напрямую (не забыть API-ключ):

```bash
grpcurl -plaintext \
  -H "x-api-key: super-secret-key" \
  -d '{"origin": "SVO", "destination": "LED"}' \
  localhost:50051 flights.FlightService/SearchFlights
```

Также есть тестовый Go-клиент `flight-client/`, который создаёт рейс через служебный метод `CreateF` (захардкоженный рейс SU100 SVO→LED) — но он ходит без API-ключа, поэтому против запущенного с ключом сервера работать не будет.

## Возможные улучшения / trade-offs

- **Тесты только на репозиторий flight-service** — HTTP-хендлеры booking-service (в т.ч. сага с компенсацией) и gRPC-слой юнит-тестами не покрыты; интеграционных тестов с реальным Postgres нет.
- **Нет graceful shutdown** — сервисы не обрабатывают SIGTERM, `http.ListenAndServe` и `grpcServer.Serve` просто убиваются; в проде нужен drain соединений.
- **Компенсация не гарантирована**: если `ReleaseReservation` после неудачного `INSERT` тоже упадёт (сеть), места останутся занятыми — нет retry/outbox, ошибка молча игнорируется.
- **API-ключ захардкожен в docker-compose.yml** (`super-secret-key`) и передаётся по plaintext gRPC (`insecure.NewCredentials`) — в проде нужны секреты и TLS/mTLS.
- **Служебный метод `CreateF` (`bebe`/`baba`)** — отладочный способ засеять данные, с захардкоженным рейсом; ему не место в контракте, данные лучше сидировать миграцией.
