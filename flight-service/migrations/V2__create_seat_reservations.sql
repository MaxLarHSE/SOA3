CREATE TABLE seat_reservations (
    id UUID PRIMARY KEY DEFAULT  gen_random_uuid(),

    booking_id UUID NOT NULL UNIQUE,

    flight_id UUID NOT NULL,
    seat_count INT NOT NULL CHECK (seat_count > 0),

    status VARCHAR NOT NULL,

    CONSTRAINT fk_reservation_flight
       FOREIGN KEY (flight_id)
           REFERENCES flights(id)
           ON DELETE CASCADE
);

