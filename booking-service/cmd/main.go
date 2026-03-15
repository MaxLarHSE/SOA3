package main

import (
	booking_service "SOA3/booking-service"
	"SOA3/booking-service/client"
	shttp "SOA3/booking-service/http"
	"log"
	"net/http"
	"os"
)

func main() {
	repo := booking_service.NewPostgresSql()
	defer repo.Close()

	flightAddr := os.Getenv("FLIGHT_SERVICE_ADDR")
	if flightAddr == "" {
		flightAddr = "localhost:50051"
	}

	flightClient, err := client.NewFlightClient(flightAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer flightClient.Close()

	handler := shttp.NewHandler(repo, flightClient)
	router := shttp.NewRouter(handler)

	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("booking-service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal(err)
	}
}
