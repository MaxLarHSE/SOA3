# SOA3

SOA3 - учебный проект с двумя сервисами для поиска рейсов и бронирования мест.

## Что внутри

- `flight-service` - gRPC-сервис рейсов. Работает с PostgreSQL, хранит рейсы и резервации мест, проверяет запросы по API-ключу.
- `booking-service` - HTTP-сервис бронирований. Принимает REST-запросы, хранит бронирования в отдельной PostgreSQL-базе и обращается к `flight-service` по gRPC.
- `proto` - protobuf-контракт `FlightService`.
- `gen/proto` - сгенерированный Go-код из protobuf.
- `docker-compose.yml` - поднимает две базы PostgreSQL, Flyway-миграции и оба сервиса.

## Как запустить

Нужны Docker и Docker Compose.

```bash
docker compose up --build
```

После запуска:

- HTTP API бронирований: `http://localhost:8080`
- gRPC API рейсов: `localhost:50051`
- PostgreSQL рейсов доступен на `localhost:5433`
- PostgreSQL бронирований доступен на `localhost:5434`

Примеры HTTP-запросов лежат в `test.http`.

## Основные HTTP-эндпоинты

- `GET /flights?origin=SVO&destination=LED` - поиск рейсов.
- `GET /flights/{id}` - получение рейса по id.
- `POST /bookings` - создание бронирования.
- `GET /bookings?user_id={id}` - список бронирований пользователя.
- `GET /bookings/{id}` - получение бронирования по id.
- `POST /bookings/{id}/cancel` - отмена бронирования.
