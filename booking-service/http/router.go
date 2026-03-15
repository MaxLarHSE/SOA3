package shttp

import (
	"github.com/go-chi/chi/v5"
	nethttp "net/http"
)

func NewRouter(h *Handler) nethttp.Handler {

	r := chi.NewRouter()

	r.Route("/flights", func(r chi.Router) {

		r.Get("/", h.GetFlights)

		r.Get("/{id}", h.GetFlight)

	})

	r.Route("/bookings", func(r chi.Router) {

		r.Post("/", h.CreateBooking)

		r.Get("/", h.GetBookingsByUserID)

		r.Get("/{id}", h.GetBookingByID)

		r.Post("/{id}/cancel", h.CancelBooking)

	})

	return r
}
