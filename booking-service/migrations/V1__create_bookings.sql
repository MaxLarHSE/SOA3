CREATE TABLE bookings (
    id UUID PRIMARY KEY DEFAULT   gen_random_uuid(),

    user_id UUID NOT NULL,

    passenger_name VARCHAR NOT NULL,
    passenger_email VARCHAR NOT NULL,

    flight_id UUID NOT NULL,

    seat_count INT NOT NULL CHECK (seat_count > 0),

    total_price NUMERIC(10,2) NOT NULL CHECK (total_price > 0),

    status VARCHAR NOT NULL
);