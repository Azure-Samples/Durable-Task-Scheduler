package main

import (
	"errors"
	"strings"

	"github.com/microsoft/durabletask-go/api"
	"github.com/microsoft/durabletask-go/task"
)

const bookingRejectedType api.ErrorType = "GoSagaBookingRejected"

type BookingRequest struct {
	RequestID                   string `json:"request_id"`
	Destination                 string `json:"destination"`
	Nights                      int    `json:"nights"`
	SimulateCarFailure          bool   `json:"simulate_car_failure"`
	SimulateCancellationFailure string `json:"simulate_cancellation_failure,omitempty"`
}

func (request BookingRequest) validate() error {
	if strings.TrimSpace(request.RequestID) == "" || strings.TrimSpace(request.Destination) == "" {
		return errors.New("booking requires a request ID and destination")
	}
	switch request.SimulateCancellationFailure {
	case "", "flight", "hotel", "car":
		return nil
	default:
		return errors.New("unknown cancellation failure service")
	}
}

type Booking struct {
	Confirmation string `json:"confirmation"`
	Service      string `json:"service"`
	Destination  string `json:"destination"`
}

type bookingRejected struct{ message string }

func (err *bookingRejected) Error() string                   { return err.message }
func (*bookingRejected) DurableTaskErrorType() api.ErrorType { return bookingRejectedType }
func (*bookingRejected) NonRetriable() bool                  { return true }

func confirmation(service, requestID string) string {
	switch service {
	case "flight":
		return "FL-" + requestID
	case "hotel":
		return "HT-" + requestID
	case "car":
		return "CR-" + requestID
	default:
		return ""
	}
}

func makeBooking(ctx task.ActivityContext, service string) (any, error) {
	var input BookingRequest
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	if err := input.validate(); err != nil {
		return nil, err
	}
	switch {
	case service == "flight" && strings.EqualFold(input.Destination, "Nowhere"):
		return nil, &bookingRejected{message: "No flights available to " + input.Destination}
	case service == "hotel" && input.Nights <= 0:
		return nil, &bookingRejected{message: "Invalid hotel booking: 0 nights"}
	case service == "car" && input.SimulateCarFailure:
		return nil, &bookingRejected{message: "No rental cars available in " + input.Destination}
	}
	// Simulation only: stable confirmation IDs model idempotency keys, not real reservations.
	return Booking{
		Confirmation: confirmation(service, input.RequestID), Service: service, Destination: input.Destination,
	}, nil
}

func bookFlight(ctx task.ActivityContext) (any, error) { return makeBooking(ctx, "flight") }
func bookHotel(ctx task.ActivityContext) (any, error)  { return makeBooking(ctx, "hotel") }
func bookCar(ctx task.ActivityContext) (any, error)    { return makeBooking(ctx, "car") }
