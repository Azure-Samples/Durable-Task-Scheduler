package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/microsoft/durabletask-go/task"
)

type SagaResult struct {
	Status        string         `json:"status"`
	Destination   string         `json:"destination"`
	Bookings      []Booking      `json:"bookings,omitempty"`
	Error         string         `json:"error,omitempty"`
	Compensations []Cancellation `json:"compensations,omitempty"`
}

func travelBookingSaga(ctx *task.OrchestrationContext) (any, error) {
	var input BookingRequest
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	result, sagaErr := executeSaga(input, func(name string, input, output any) error {
		options := []task.CallActivityOption{task.WithActivityInput(input)}
		if strings.HasPrefix(name, "GoSagaCancel") {
			options = append(options, task.WithActivityRetryPolicy(compensationRetryPolicy()))
		}
		return ctx.CallActivity(name, options...).Await(output)
	})
	statusErr := ctx.SetCustomStatusValue(result)
	if sagaErr != nil || statusErr != nil {
		return nil, errors.Join(sagaErr, statusErr)
	}
	return result, nil
}

func executeSaga(input BookingRequest, call func(string, any, any) error) (SagaResult, error) {
	if err := input.validate(); err != nil {
		return SagaResult{}, err
	}
	steps := []struct {
		service string
		book    string
		cancel  string
	}{
		{"flight", bookFlightName, cancelFlightName},
		{"hotel", bookHotelName, cancelHotelName},
		{"car", bookCarName, cancelCarName},
	}
	result := SagaResult{Status: "success", Destination: input.Destination}
	var bookingErr error
	for _, step := range steps {
		var booking Booking
		bookingErr = call(step.book, input, &booking)
		if bookingErr != nil {
			break
		}
		want := Booking{Service: step.service, Destination: input.Destination, Confirmation: confirmation(step.service, input.RequestID)}
		if booking != want {
			bookingErr = fmt.Errorf("invalid %s booking receipt: %+v", step.service, booking)
			break
		}
		result.Bookings = append(result.Bookings, booking)
	}
	if bookingErr == nil {
		return result, nil
	}

	result.Status = "failed"
	result.Error = failureMessage(bookingErr)
	var cancellationErrors []error
	var cancellationMessages []string
	// Undo only completed bookings, in reverse order; attempt every compensation.
	for i := len(result.Bookings) - 1; i >= 0; i-- {
		booking := result.Bookings[i]
		var cancelled Cancellation
		err := call(steps[i].cancel, CancellationInput{
			Booking: booking, SimulateFailure: input.SimulateCancellationFailure == booking.Service,
		}, &cancelled)
		want := Cancellation{Service: booking.Service, Confirmation: booking.Confirmation, Status: "cancelled"}
		if err == nil && cancelled != want {
			err = fmt.Errorf("invalid %s cancellation receipt: %+v", booking.Service, cancelled)
		}
		if err != nil {
			message := failureMessage(err)
			result.Compensations = append(result.Compensations, Cancellation{
				Service: booking.Service, Confirmation: booking.Confirmation, Status: "failed", Error: message,
			})
			cancellationErrors = append(cancellationErrors, err)
			cancellationMessages = append(cancellationMessages, booking.Service+": "+message)
			continue
		}
		result.Compensations = append(result.Compensations, cancelled)
	}
	if len(cancellationErrors) != 0 {
		result.Status = "compensation_failed"
		return result, &compensationFailure{
			bookingFailure: result.Error, failures: cancellationMessages,
			cause: errors.Join(append([]error{bookingErr}, cancellationErrors...)...),
		}
	}
	if !isBookingRejection(bookingErr) {
		return result, fmt.Errorf("unexpected booking failure; prior bookings compensated: %w", bookingErr)
	}
	return result, nil
}
