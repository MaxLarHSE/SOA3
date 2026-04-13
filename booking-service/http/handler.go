package shttp

import (
	booking_service "SOA3/booking-service"
	"SOA3/booking-service/client"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	bookingRepo  *booking_service.BookingRepo
	flightClient *client.FlightClient
}

// asdasd asdasdasd
func NewHandler(
	bookingRepo *booking_service.BookingRepo,
	flightClient *client.FlightClient,
) *Handler {
	return &Handler{
		bookingRepo:  bookingRepo,
		flightClient: flightClient,
	}
}
func (h *Handler) GetFlights(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	origin := r.URL.Query().Get("origin")
	destination := r.URL.Query().Get("destination")
	dateStr := r.URL.Query().Get("date")

	if origin == "" || destination == "" {
		http.Error(w, "origin and destination are required", http.StatusBadRequest)
		return
	}

	var datePb *timestamppb.Timestamp
	if dateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			http.Error(w, "invalid date format, use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		datePb = timestamppb.New(parsedDate)
	}

	flights, err := h.flightClient.SearchFlights(ctx, origin, destination, datePb)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(flights)
}

// GET /flights/{id}
func (h *Handler) GetFlight(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "flight id is required", http.StatusBadRequest)
		return
	}

	flight, err := h.flightClient.GetFlight(ctx, id)
	if err != nil {
		http.Error(w, "flight not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(flight)
}

// POST /bookings
func (h *Handler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" || req.FlightID == "" || req.PassengerName == "" || req.PassengerEmail == "" || req.SeatCount <= 0 {
		http.Error(w, "invalid request fields", http.StatusBadRequest)
		return
	}

	flight, err := h.flightClient.GetFlight(ctx, req.FlightID)
	if err != nil {
		http.Error(w, "flight not found", http.StatusNotFound)
		return
	}

	bookingID := uuid.NewString()

	err = h.flightClient.ReserveSeats(ctx, bookingID, req.FlightID, req.SeatCount)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	totalPrice := float64(req.SeatCount) * flight.Price

	err = h.bookingRepo.CreateBooking(
		ctx,
		bookingID,
		req.UserID,
		req.FlightID,
		req.PassengerName,
		req.PassengerEmail,
		req.SeatCount,
		totalPrice,
	)
	if err != nil {
		_ = h.flightClient.ReleaseReservation(ctx, bookingID)
		http.Error(w, "failed to create booking", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":          bookingID,
		"status":      "CONFIRMED",
		"total_price": totalPrice,
	})
}

// GET /bookings/{id}
func (h *Handler) GetBookingByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "booking id is required", http.StatusBadRequest)
		return
	}

	booking, err := h.bookingRepo.GetBookingByID(ctx, id)
	if err != nil {
		http.Error(w, "booking not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(booking)
}

// GET /bookings?user_id=...
func (h *Handler) GetBookingsByUserID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	bookings, err := h.bookingRepo.GetBookingsByUserID(ctx, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(bookings)
}

// POST /bookings/{id}/cancel
func (h *Handler) CancelBooking(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "booking id is required", http.StatusBadRequest)
		return
	}

	booking, err := h.bookingRepo.GetBookingByID(ctx, id)
	if err != nil {
		http.Error(w, "booking not found", http.StatusNotFound)
		return
	}

	if booking.Status != "CONFIRMED" {
		http.Error(w, "booking is not confirmed", http.StatusBadRequest)
		return
	}

	if err := h.flightClient.ReleaseReservation(ctx, id); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	if err := h.bookingRepo.CancelBooking(ctx, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":     id,
		"status": "CANCELLED",
	})
}
