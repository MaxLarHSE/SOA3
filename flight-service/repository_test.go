package flight_service

import (
	pb "SOA3/gen/proto"
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func newMockRepo(t *testing.T) (*FlightRepo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return &FlightRepo{db: db}, mock
}

const (
	testFlightID  = "f0000000-0000-0000-0000-000000000001"
	testBookingID = "b0000000-0000-0000-0000-000000000001"
)

var (
	reSelectSeats       = regexp.QuoteMeta(`SELECT available_seats FROM flights WHERE id = $1 FOR UPDATE`)
	reUpdateSeatsMinus  = regexp.QuoteMeta(`UPDATE flights`) + `\s+SET available_seats = available_seats - \$1`
	reUpdateSeatsPlus   = regexp.QuoteMeta(`UPDATE flights`) + `\s+SET available_seats = available_seats \+ \$1`
	reInsertReservation = regexp.QuoteMeta(`INSERT INTO seat_reservations`)
	reSelectReservation = `SELECT flight_id, seat_count\s+FROM seat_reservations\s+WHERE booking_id = \$1 AND status = 'ACTIVE'\s+FOR UPDATE`
	reReleaseUpdate     = regexp.QuoteMeta(`UPDATE seat_reservations`) + `\s+SET status = 'RELEASED'`
)

func TestReserveSeats_Success(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectBegin()
	mock.ExpectQuery(reSelectSeats).
		WithArgs(testFlightID).
		WillReturnRows(sqlmock.NewRows([]string{"available_seats"}).AddRow(10))
	mock.ExpectExec(reUpdateSeatsMinus).
		WithArgs(int32(3), testFlightID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(reInsertReservation).
		WithArgs(testBookingID, testFlightID, int32(3)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := repo.ReserveSeats(context.Background(), testFlightID, 3, testBookingID); err != nil {
		t.Fatalf("ReserveSeats: unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestReserveSeats_NotEnoughSeats_RollsBack(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectBegin()
	mock.ExpectQuery(reSelectSeats).
		WithArgs(testFlightID).
		WillReturnRows(sqlmock.NewRows([]string{"available_seats"}).AddRow(1))
	mock.ExpectRollback()

	err := repo.ReserveSeats(context.Background(), testFlightID, 2, testBookingID)
	if !errors.Is(err, ErrNotEnoughSeats) {
		t.Fatalf("ReserveSeats: want ErrNotEnoughSeats, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestReserveSeats_DuplicateBooking_RollsBack(t *testing.T) {
	repo, mock := newMockRepo(t)

	uniqueViolation := errors.New(`pq: duplicate key value violates unique constraint "seat_reservations_booking_id_key"`)

	mock.ExpectBegin()
	mock.ExpectQuery(reSelectSeats).
		WithArgs(testFlightID).
		WillReturnRows(sqlmock.NewRows([]string{"available_seats"}).AddRow(10))
	mock.ExpectExec(reUpdateSeatsMinus).
		WithArgs(int32(2), testFlightID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(reInsertReservation).
		WithArgs(testBookingID, testFlightID, int32(2)).
		WillReturnError(uniqueViolation)
	mock.ExpectRollback()

	err := repo.ReserveSeats(context.Background(), testFlightID, 2, testBookingID)
	if !errors.Is(err, uniqueViolation) {
		t.Fatalf("ReserveSeats: want unique violation error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestReserveSeats_FlightNotFound_RollsBack(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectBegin()
	mock.ExpectQuery(reSelectSeats).
		WithArgs(testFlightID).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	err := repo.ReserveSeats(context.Background(), testFlightID, 1, testBookingID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("ReserveSeats: want sql.ErrNoRows, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestReleaseReservation_Success(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectBegin()
	mock.ExpectQuery(reSelectReservation).
		WithArgs(testBookingID).
		WillReturnRows(sqlmock.NewRows([]string{"flight_id", "seat_count"}).AddRow(testFlightID, 2))
	mock.ExpectExec(reUpdateSeatsPlus).
		WithArgs(int32(2), testFlightID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(reReleaseUpdate).
		WithArgs(testBookingID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := repo.ReleaseReservation(context.Background(), testBookingID); err != nil {
		t.Fatalf("ReleaseReservation: unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestReleaseReservation_NotActive_RollsBack(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectBegin()
	mock.ExpectQuery(reSelectReservation).
		WithArgs(testBookingID).
		WillReturnRows(sqlmock.NewRows([]string{"flight_id", "seat_count"}))
	mock.ExpectRollback()

	err := repo.ReleaseReservation(context.Background(), testBookingID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("ReleaseReservation: want sql.ErrNoRows, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestMapFlightStatus(t *testing.T) {
	cases := []struct {
		in   string
		want pb.FlightStatus
	}{
		{"SCHEDULED", pb.FlightStatus_SCHEDULED},
		{"DEPARTED", pb.FlightStatus_DEPARTED},
		{"CANCELLED", pb.FlightStatus_CANCELLED},
		{"COMPLETED", pb.FlightStatus_COMPLETED},
		{"", pb.FlightStatus_FLIGHT_STATUS_UNSPECIFIED},
		{"garbage", pb.FlightStatus_FLIGHT_STATUS_UNSPECIFIED},
	}
	for _, c := range cases {
		t.Run("in="+c.in, func(t *testing.T) {
			if got := mapFlightStatus(c.in); got != c.want {
				t.Errorf("mapFlightStatus(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}
