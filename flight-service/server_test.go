package flight_service

import (
	pb "SOA3/gen/proto"
	"context"
	"database/sql"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeRepo struct {
	getFlightByID      func(ctx context.Context, id string) (*pb.Flight, error)
	searchFlights      func(ctx context.Context, origin, destination string, date *timestamppb.Timestamp) ([]*pb.Flight, error)
	reserveSeats       func(ctx context.Context, flightID string, seatCount int32, bookingID string) error
	releaseReservation func(ctx context.Context, bookingID string) error
	createFlight       func(ctx context.Context, f *pb.Flight) (string, error)
}

func (f *fakeRepo) GetFlightByID(ctx context.Context, id string) (*pb.Flight, error) {
	return f.getFlightByID(ctx, id)
}
func (f *fakeRepo) SearchFlights(ctx context.Context, origin, destination string, date *timestamppb.Timestamp) ([]*pb.Flight, error) {
	return f.searchFlights(ctx, origin, destination, date)
}
func (f *fakeRepo) ReserveSeats(ctx context.Context, flightID string, seatCount int32, bookingID string) error {
	return f.reserveSeats(ctx, flightID, seatCount, bookingID)
}
func (f *fakeRepo) ReleaseReservation(ctx context.Context, bookingID string) error {
	return f.releaseReservation(ctx, bookingID)
}
func (f *fakeRepo) CreateFlight(ctx context.Context, fl *pb.Flight) (string, error) {
	return f.createFlight(ctx, fl)
}

func wantCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("want gRPC status error, got %v", err)
	}
	if st.Code() != want {
		t.Fatalf("want code %v, got %v (%v)", want, st.Code(), err)
	}
}

func TestServerGetFlight(t *testing.T) {
	cases := []struct {
		name     string
		repoErr  error
		wantCode codes.Code
	}{
		{"not found", sql.ErrNoRows, codes.NotFound},
		{"internal", errors.New("db is down"), codes.Internal},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := NewServer(&fakeRepo{
				getFlightByID: func(ctx context.Context, id string) (*pb.Flight, error) {
					return nil, c.repoErr
				},
			})
			_, err := s.GetFlight(context.Background(), &pb.FlightRequest{Id: testFlightID})
			wantCode(t, err, c.wantCode)
		})
	}

	t.Run("success", func(t *testing.T) {
		want := &pb.Flight{Id: testFlightID, FlightNumber: "SU100"}
		s := NewServer(&fakeRepo{
			getFlightByID: func(ctx context.Context, id string) (*pb.Flight, error) {
				if id != testFlightID {
					t.Errorf("unexpected id %q", id)
				}
				return want, nil
			},
		})
		got, err := s.GetFlight(context.Background(), &pb.FlightRequest{Id: testFlightID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Id != want.Id || got.FlightNumber != want.FlightNumber {
			t.Errorf("got %v, want %v", got, want)
		}
	})
}

func TestServerReserveSeats(t *testing.T) {
	cases := []struct {
		name     string
		repoErr  error
		wantCode codes.Code
	}{
		{"flight not found", sql.ErrNoRows, codes.NotFound},
		{"not enough seats", ErrNotEnoughSeats, codes.ResourceExhausted},
		{"internal", errors.New("db is down"), codes.Internal},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := NewServer(&fakeRepo{
				reserveSeats: func(ctx context.Context, flightID string, seatCount int32, bookingID string) error {
					return c.repoErr
				},
			})
			_, err := s.ReserveSeats(context.Background(), &pb.ReserveRequest{
				BookingId: testBookingID,
				FlightId:  testFlightID,
				SeatCount: 2,
			})
			wantCode(t, err, c.wantCode)
		})
	}

	t.Run("success", func(t *testing.T) {
		s := NewServer(&fakeRepo{
			reserveSeats: func(ctx context.Context, flightID string, seatCount int32, bookingID string) error {
				if flightID != testFlightID || bookingID != testBookingID || seatCount != 2 {
					t.Errorf("unexpected args: %q %q %d", flightID, bookingID, seatCount)
				}
				return nil
			},
		})
		reply, err := s.ReserveSeats(context.Background(), &pb.ReserveRequest{
			BookingId: testBookingID,
			FlightId:  testFlightID,
			SeatCount: 2,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reply.Success {
			t.Error("want Success = true")
		}
	})
}

func TestServerReleaseReservation(t *testing.T) {
	cases := []struct {
		name     string
		repoErr  error
		wantCode codes.Code
	}{
		{"reservation not found", sql.ErrNoRows, codes.NotFound},
		{"not active", ErrReservationNotActive, codes.FailedPrecondition},
		{"internal", errors.New("db is down"), codes.Internal},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := NewServer(&fakeRepo{
				releaseReservation: func(ctx context.Context, bookingID string) error {
					return c.repoErr
				},
			})
			_, err := s.ReleaseReservation(context.Background(), &pb.ReleaseRequest{BookingId: testBookingID})
			wantCode(t, err, c.wantCode)
		})
	}

	t.Run("success", func(t *testing.T) {
		s := NewServer(&fakeRepo{
			releaseReservation: func(ctx context.Context, bookingID string) error {
				if bookingID != testBookingID {
					t.Errorf("unexpected bookingID %q", bookingID)
				}
				return nil
			},
		})
		reply, err := s.ReleaseReservation(context.Background(), &pb.ReleaseRequest{BookingId: testBookingID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reply.Success {
			t.Error("want Success = true")
		}
	})
}
