package main

import (
	"errors"
	"strings"
	"time"

	"github.com/microsoft/durabletask-go/api"
	"github.com/microsoft/durabletask-go/task"
)

const (
	cancellationErrorType api.ErrorType = "GoSagaCancellationUnavailable"
	compensationErrorType api.ErrorType = "GoSagaCompensationFailed"
)

type CancellationInput struct {
	Booking         Booking `json:"booking"`
	SimulateFailure bool    `json:"simulate_failure"`
}

type Cancellation struct {
	Service      string `json:"service"`
	Confirmation string `json:"confirmation"`
	Status       string `json:"status"`
	Error        string `json:"error,omitempty"`
}

func cancelBooking(ctx task.ActivityContext, service string) (any, error) {
	var input CancellationInput
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	if input.Booking.Service != service || input.Booking.Confirmation == "" {
		return nil, errors.New("cancellation does not identify a matching booking")
	}
	if input.SimulateFailure {
		return nil, &cancellationUnavailable{service: service}
	}
	// Simulation only: no booking provider is contacted.
	return Cancellation{Service: service, Confirmation: input.Booking.Confirmation, Status: "cancelled"}, nil
}

func cancelFlight(ctx task.ActivityContext) (any, error) { return cancelBooking(ctx, "flight") }
func cancelHotel(ctx task.ActivityContext) (any, error)  { return cancelBooking(ctx, "hotel") }
func cancelCar(ctx task.ActivityContext) (any, error)    { return cancelBooking(ctx, "car") }

func compensationRetryPolicy() *task.RetryPolicy {
	return &task.RetryPolicy{
		MaxAttempts: 3, InitialRetryInterval: 100 * time.Millisecond, BackoffCoefficient: 2,
		MaxRetryInterval: 500 * time.Millisecond, RetryTimeout: 10 * time.Second,
	}
}

type cancellationUnavailable struct{ service string }

func (err *cancellationUnavailable) Error() string {
	return "simulated " + err.service + " cancellation outage"
}
func (*cancellationUnavailable) DurableTaskErrorType() api.ErrorType { return cancellationErrorType }

type compensationFailure struct {
	bookingFailure string
	failures       []string
	cause          error
}

func (err *compensationFailure) Error() string {
	return "compensation incomplete after " + err.bookingFailure + ": " + strings.Join(err.failures, "; ")
}
func (err *compensationFailure) Unwrap() error                   { return err.cause }
func (*compensationFailure) DurableTaskErrorType() api.ErrorType { return compensationErrorType }

func failureMessage(err error) string {
	var remote *task.TaskFailedError
	if errors.As(err, &remote) {
		for details := remote.FailureDetails; details != nil; details = details.InnerFailure {
			if details.ErrorType == bookingRejectedType || details.ErrorType == cancellationErrorType {
				return details.ErrorMessage
			}
		}
	}
	return err.Error()
}

func isBookingRejection(err error) bool {
	var rejected *bookingRejected
	if errors.As(err, &rejected) {
		return true
	}
	var remote *task.TaskFailedError
	return errors.As(err, &remote) && remote.FailureDetails.IsCausedBy(bookingRejectedType)
}
