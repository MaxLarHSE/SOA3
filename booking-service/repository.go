package booking_service

import (
	"context"
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
)

type Booking struct {
	ID             string  `json:"id"`
	UserID         string  `json:"user_id"`
	FlightID       string  `json:"flight_id"`
	PassengerName  string  `json:"passenger_name"`
	PassengerEmail string  `json:"passenger_email"`
	SeatCount      int32   `json:"seat_count"`
	TotalPrice     float64 `json:"total_price"`
	Status         string  `json:"status"`
}
type BookingRepo struct {
	db *sql.DB
}

func NewPostgresSql() *BookingRepo {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("Null flight data base env")
	}
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Ошибка при создании пула соединений: %v", err)
	}
	//defer db.Close() НАПИСАТЬ В МЕЙН

	if err := db.Ping(); err != nil {
		log.Fatalf("База данных недоступна: %v", err)
	}
	log.Println("Успешное подключение к PostgreSQL!")
	return &BookingRepo{db: db}
}
func (r *BookingRepo) Close() {
	r.db.Close()
}
func (r *BookingRepo) CreateBooking(ctx context.Context, id string, userID, flightID, passengerName, passengerEmail string, seatCount int32, totalPrice float64) error {
	query := `
	INSERT INTO bookings (
		id,
		user_id,
		flight_id,
		passenger_name,
		passenger_email,
		seat_count,
		total_price,
		status
	)
	VALUES ($1,$2,$3,$4,$5,$6,$7,'CONFIRMED')
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		id,
		userID,
		flightID,
		passengerName,
		passengerEmail,
		seatCount,
		totalPrice,
	)

	return err
}
func (r *BookingRepo) GetBookingByID(ctx context.Context, id string) (*Booking, error) {
	query := `
	SELECT id,
	       user_id,
	       flight_id,
	       passenger_name,
	       passenger_email,
	       seat_count,
	       total_price,
	       status
	FROM bookings
	WHERE id = $1
	`

	var b Booking

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&b.ID,
		&b.UserID,
		&b.FlightID,
		&b.PassengerName,
		&b.PassengerEmail,
		&b.SeatCount,
		&b.TotalPrice,
		&b.Status,
	)

	if err != nil {
		return nil, err
	}

	return &b, nil
}
func (r *BookingRepo) GetBookingsByUserID(ctx context.Context, userID string) ([]*Booking, error) {
	query := `
	SELECT id,
	       user_id,
	       flight_id,
	       passenger_name,
	       passenger_email,
	       seat_count,
	       total_price,
	       status
	FROM bookings
	WHERE user_id = $1
	ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var bookings []*Booking

	for rows.Next() {

		var b Booking

		err := rows.Scan(
			&b.ID,
			&b.UserID,
			&b.FlightID,
			&b.PassengerName,
			&b.PassengerEmail,
			&b.SeatCount,
			&b.TotalPrice,
			&b.Status,
		)

		if err != nil {
			return nil, err
		}

		bookings = append(bookings, &b)
	}

	return bookings, nil
}
func (r *BookingRepo) CancelBooking(ctx context.Context, id string) error {
	query := `
	UPDATE bookings
	SET status = 'CANCELLED'
	WHERE id = $1
	`

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}
