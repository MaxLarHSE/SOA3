package main

import (
	pb "SOA3/gen/proto"
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient("127.0.0.1:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	client := pb.NewFlightServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	baba, err := client.CreateF(ctx, &pb.Bebe{})
	if err != nil {
		log.Fatal(err)
	}
	_, err = client.GetFlight(ctx, &pb.FlightRequest{Id: baba.Id})
	if err != nil {
		log.Fatal(err)
	}
}
