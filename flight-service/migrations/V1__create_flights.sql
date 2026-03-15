CREATE TABLE flights (
    id UUID PRIMARY KEY DEFAULT  gen_random_uuid(),

    flight_number VARCHAR NOT NULL,

    origin VARCHAR(3) NOT NULL,
    destination VARCHAR(3) NOT NULL,

    departure_time TIMESTAMP NOT NULL,
    arrival_time TIMESTAMP NOT NULL,

    total_seats INT NOT NULL CHECK (total_seats > 0),
    available_seats INT NOT NULL CHECK (available_seats >= 0),

    price NUMERIC(10,2) NOT NULL CHECK (price > 0),

    status VARCHAR NOT NULL,

    CONSTRAINT flights_unique_flight UNIQUE (flight_number, departure_time),
    CONSTRAINT seats_valid CHECK (available_seats <= total_seats),
    CONSTRAINT different_airports CHECK (origin <> destination),
    CONSTRAINT valid_time CHECK (arrival_time > departure_time)
);