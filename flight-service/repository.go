package flight_service

import (
	pb "SOA3/gen/proto"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	ErrNotEnoughSeats       = errors.New("not enough seats")
	ErrReservationExists    = errors.New("reservation already exists")
	ErrReservationNotActive = errors.New("reservation is not active")
)

type FlightRepo struct {
	db *sql.DB
}

func (r *FlightRepo) Close() {
	r.db.Close()
}

func (r *FlightRepo) GetFlightByID(ctx context.Context, id string) (*pb.Flight, error) {
	query := `
	SELECT id,
	       flight_number,
	       origin,
	       destination,
	       departure_time,
	       arrival_time,
	       total_seats,
	       available_seats,
	       price,
	       status
	FROM flights
	WHERE id = $1
`
	var flightID string
	var flightNumber string
	var origin string
	var destination string
	var departureTime time.Time
	var arrivalTime time.Time
	var totalSeats int32
	var availableSeats int32
	var price float64
	var status string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&flightID,
		&flightNumber,
		&origin,
		&destination,
		&departureTime,
		&arrivalTime,
		&totalSeats,
		&availableSeats,
		&price,
		&status,
	)
	if err != nil {
		return nil, err
	}
	return &pb.Flight{
		Id:             flightID,
		FlightNumber:   flightNumber,
		Origin:         origin,
		Destination:    destination,
		DepartureTime:  timestamppb.New(departureTime),
		ArrivalTime:    timestamppb.New(arrivalTime),
		TotalSeats:     totalSeats,
		AvailableSeats: availableSeats,
		Price:          price,
		Status:         mapFlightStatus(status),
	}, nil
}
func mapFlightStatus(s string) pb.FlightStatus {
	switch s {
	case "SCHEDULED":
		return pb.FlightStatus_SCHEDULED
	case "DEPARTED":
		return pb.FlightStatus_DEPARTED
	case "CANCELLED":
		return pb.FlightStatus_CANCELLED
	case "COMPLETED":
		return pb.FlightStatus_COMPLETED
	default:
		return pb.FlightStatus_FLIGHT_STATUS_UNSPECIFIED
	}
}
func (r *FlightRepo) CreateFlight(ctx context.Context, f *pb.Flight) (string, error) {
	var id string

	err := r.db.QueryRowContext(ctx, `
		INSERT INTO flights (
			flight_number,
			origin,
			destination,
			departure_time,
			arrival_time,
			total_seats,
			available_seats,
			price,
			status
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id
`,
		f.FlightNumber,
		f.Origin,
		f.Destination,
		f.DepartureTime.AsTime(),
		f.ArrivalTime.AsTime(),
		f.TotalSeats,
		f.AvailableSeats,
		f.Price,
		f.Status.String(),
	).Scan(&id)

	return id, err
}
func NewPostgresSql() *FlightRepo {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("Null flight data base env")
	}
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Ошибка при создании пула соединений: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("База данных недоступна: %v", err)
	}
	log.Println("Успешное подключение к PostgreSQL!")
	return &FlightRepo{db: db}
}
func (r *FlightRepo) SearchFlights(ctx context.Context, origin, destination string, date *timestamppb.Timestamp) ([]*pb.Flight, error) {
	var (
		rows *sql.Rows
		err  error
	)

	if date != nil && !date.AsTime().IsZero() {
		query := `
		SELECT id,
		       flight_number,
		       origin,
		       destination,
		       departure_time,
		       arrival_time,
		       total_seats,
		       available_seats,
		       price,
		       status
		FROM flights
		WHERE origin = $1
		  AND destination = $2
		  AND status = 'SCHEDULED'
		  AND DATE(departure_time) = DATE($3)
		ORDER BY departure_time
		`
		rows, err = r.db.QueryContext(ctx, query, origin, destination, date.AsTime())
	} else {
		query := `
		SELECT id,
		       flight_number,
		       origin,
		       destination,
		       departure_time,
		       arrival_time,
		       total_seats,
		       available_seats,
		       price,
		       status
		FROM flights
		WHERE origin = $1
		  AND destination = $2
		  AND status = 'SCHEDULED'
		ORDER BY departure_time
		`
		rows, err = r.db.QueryContext(ctx, query, origin, destination)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flights []*pb.Flight
	for rows.Next() {
		var flightID string
		var flightNumber string
		var origin string
		var destination string
		var departureTime time.Time
		var arrivalTime time.Time
		var totalSeats int32
		var availableSeats int32
		var price float64
		var status string

		err := rows.Scan(
			&flightID,
			&flightNumber,
			&origin,
			&destination,
			&departureTime,
			&arrivalTime,
			&totalSeats,
			&availableSeats,
			&price,
			&status,
		)
		if err != nil {
			return nil, err
		}

		flights = append(flights, &pb.Flight{
			Id:             flightID,
			FlightNumber:   flightNumber,
			Origin:         origin,
			Destination:    destination,
			DepartureTime:  timestamppb.New(departureTime),
			ArrivalTime:    timestamppb.New(arrivalTime),
			TotalSeats:     totalSeats,
			AvailableSeats: availableSeats,
			Price:          price,
			Status:         mapFlightStatus(status),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return flights, nil
}

func (r *FlightRepo) ReserveSeats(ctx context.Context, flightID string, seatCount int32, bookingID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var availableSeats int32
	err = tx.QueryRowContext(ctx,
		`SELECT available_seats FROM flights WHERE id = $1 FOR UPDATE`,
		flightID,
	).Scan(&availableSeats)
	if err != nil {
		return err
	}

	if availableSeats < seatCount {
		return fmt.Errorf("not enough seats")
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE flights
		 SET available_seats = available_seats - $1
		 WHERE id = $2`,
		seatCount, flightID,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO seat_reservations (booking_id, flight_id, seat_count, status)
		 VALUES ($1, $2, $3, 'ACTIVE')`,
		bookingID, flightID, seatCount,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *FlightRepo) ReleaseReservation(ctx context.Context, bookingID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var flightID string
	var seatCount int32

	err = tx.QueryRowContext(ctx,
		`SELECT flight_id, seat_count
		 FROM seat_reservations
		 WHERE booking_id = $1 AND status = 'ACTIVE'
		 FOR UPDATE`,
		bookingID,
	).Scan(&flightID, &seatCount)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE flights
		 SET available_seats = available_seats + $1
		 WHERE id = $2`,
		seatCount, flightID,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE seat_reservations
		 SET status = 'RELEASED'
		 WHERE booking_id = $1 AND status = 'ACTIVE'`,
		bookingID,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}
