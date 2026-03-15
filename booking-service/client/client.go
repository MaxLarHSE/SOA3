package client

import (
	pb "SOA3/gen/proto"
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type FlightClient struct {
	conn   *grpc.ClientConn
	client pb.FlightServiceClient
}

func (c *FlightClient) Close() error {
	return c.conn.Close()
}
func NewFlightClient(addr string) (*FlightClient, error) {

	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		return nil, err
	}

	return &FlightClient{
		conn:   conn,
		client: pb.NewFlightServiceClient(conn),
	}, nil
}

func (c *FlightClient) GetFlight(ctx context.Context, id string) (*pb.Flight, error) {
	f, err := c.client.GetFlight(ctx, &pb.FlightRequest{Id: id})

	return f, err
}
func (c *FlightClient) SearchFlights(ctx context.Context, origin, destination string, date *timestamppb.Timestamp) ([]*pb.Flight, error) {
	resp, err := c.client.SearchFlights(ctx, &pb.FlightsRequest{
		Origin:      origin,
		Destination: destination,
		Date:        date,
	})
	if err != nil {
		return nil, err
	}
	return resp.Flights, nil
}

func (c *FlightClient) ReserveSeats(ctx context.Context, bookingID, flightID string, seatCount int32) error {
	_, err := c.client.ReserveSeats(ctx, &pb.ReserveRequest{
		BookingId: bookingID,
		FlightId:  flightID,
		SeatCount: seatCount,
	})
	return err
}

func (c *FlightClient) ReleaseReservation(ctx context.Context, bookingID string) error {
	_, err := c.client.ReleaseReservation(ctx, &pb.ReleaseRequest{
		BookingId: bookingID,
	})
	return err
}
