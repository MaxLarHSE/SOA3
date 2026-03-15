package main

import (
	flight_service "SOA3/flight-service"
	pb "SOA3/gen/proto"
	"log"
	"net"

	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("listen")
	grpcServer := grpc.NewServer()
	repo := flight_service.NewPostgresSql()
	defer repo.Close()
	server := flight_service.NewServer(repo)
	pb.RegisterFlightServiceServer(grpcServer, server)
	log.Println("serve")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
