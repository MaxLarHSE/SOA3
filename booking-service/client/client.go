package client

import (
	pb "SOA3/gen/proto"
	"context"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type FlightClient struct {
	conn   *grpc.ClientConn
	client pb.FlightServiceClient
	apiKey string
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
		apiKey: os.Getenv("FLIGHT_SERVICE_API_KEY"),
	}, nil
}
func (c *FlightClient) withAuth(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "x-api-key", c.apiKey)
}
func (c *FlightClient) GetFlight(ctx context.Context, id string) (*pb.Flight, error) {
	f, err := c.client.GetFlight(c.withAuth(ctx), &pb.FlightRequest{Id: id})

	return f, err
}
func (c *FlightClient) SearchFlights(ctx context.Context, origin, destination string, date *timestamppb.Timestamp) ([]*pb.Flight, error) {
	resp, err := c.client.SearchFlights(c.withAuth(ctx), &pb.FlightsRequest{
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
	_, err := c.client.ReserveSeats(c.withAuth(ctx), &pb.ReserveRequest{
		BookingId: bookingID,
		FlightId:  flightID,
		SeatCount: seatCount,
	})
	return err
}

func (c *FlightClient) ReleaseReservation(ctx context.Context, bookingID string) error {
	_, err := c.client.ReleaseReservation(c.withAuth(ctx), &pb.ReleaseRequest{
		BookingId: bookingID,
	})
	return err
}
