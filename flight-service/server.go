package flight_service

import (
	pb "SOA3/gen/proto"
	"context"
	"database/sql"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	pb.UnimplementedFlightServiceServer
	repo *FlightRepo
}

func NewServer(repo *FlightRepo) *Server {
	return &Server{
		repo: repo,
	}
}

func (s *Server) GetFlight(ctx context.Context, req *pb.FlightRequest) (*pb.Flight, error) {
	f, err := s.repo.GetFlightByID(ctx, req.Id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "flight not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return f, nil
}

func (s *Server) SearchFlights(ctx context.Context, req *pb.FlightsRequest) (*pb.FlightsReply, error) {
	flights, err := s.repo.SearchFlights(ctx, req.Origin, req.Destination, req.Date)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.FlightsReply{
		Flights: flights,
	}, nil
}

func (s *Server) ReserveSeats(ctx context.Context, req *pb.ReserveRequest) (*pb.ReserveReply, error) {
	err := s.repo.ReserveSeats(ctx, req.FlightId, req.SeatCount, req.BookingId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "flight not found")
		}
		if err.Error() == "not enough seats" {
			return nil, status.Error(codes.ResourceExhausted, "not enough seats")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.ReserveReply{
		Success: true,
	}, nil
}

func (s *Server) ReleaseReservation(ctx context.Context, req *pb.ReleaseRequest) (*pb.ReleaseReply, error) {
	err := s.repo.ReleaseReservation(ctx, req.BookingId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "reservation not found")
		}
		if err.Error() == "reservation is not active" {
			return nil, status.Error(codes.FailedPrecondition, "reservation is not active")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.ReleaseReply{
		Success: true,
	}, nil
}

func (s *Server) CreateF(ctx context.Context, req *pb.Bebe) (*pb.Baba, error) {
	id, err := s.repo.CreateFlight(ctx, &pb.Flight{
		FlightNumber:   "SU100",
		Origin:         "SVO",
		Destination:    "LED",
		DepartureTime:  timestamppb.New(time.Now().Add(2 * time.Hour)),
		ArrivalTime:    timestamppb.New(time.Now().Add(4 * time.Hour)),
		TotalSeats:     180,
		AvailableSeats: 180,
		Price:          12500.50,
		Status:         mapFlightStatus("SCHEDULED"),
	})
	return &pb.Baba{Id: id}, err
}
