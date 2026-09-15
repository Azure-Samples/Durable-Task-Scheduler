package main

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/microsoft/durabletask-go/api"
	"github.com/microsoft/durabletask-go/task"
)

type activityInput []byte

func (input activityInput) GetInput(target any) error { return json.Unmarshal(input, target) }
func (activityInput) Context() context.Context        { return context.Background() }

func invokeFixture(t *testing.T, calls *[]string) func(string, any, any) error {
	t.Helper()
	activities := map[string]task.Activity{
		bookFlightName: bookFlight, bookHotelName: bookHotel, bookCarName: bookCar,
		cancelFlightName: cancelFlight, cancelHotelName: cancelHotel, cancelCarName: cancelCar,
	}
	return func(name string, input, target any) error {
		*calls = append(*calls, name)
		data, err := json.Marshal(input)
		if err != nil {
			return err
		}
		output, err := activities[name](activityInput(data))
		if err != nil {
			var provider api.DurableTaskErrorTypeProvider
			if errors.As(err, &provider) {
				return &task.TaskFailedError{TaskName: name, FailureDetails: &api.FailureDetails{
					ErrorType: provider.DurableTaskErrorType(), ErrorMessage: err.Error(),
				}}
			}
			return err
		}
		data, err = json.Marshal(output)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, target)
	}
}

func TestSagaSuccessAndRollbackOrder(t *testing.T) {
	for _, scenario := range []struct {
		input             BookingRequest
		status            string
		wantCalls         []string
		wantCompensations []string
	}{
		{BookingRequest{RequestID: "trip", Destination: "Paris", Nights: 5}, "success",
			[]string{bookFlightName, bookHotelName, bookCarName}, nil},
		{BookingRequest{RequestID: "trip", Destination: "Tokyo", Nights: 3, SimulateCarFailure: true}, "failed",
			[]string{bookFlightName, bookHotelName, bookCarName, cancelHotelName, cancelFlightName}, []string{"hotel", "flight"}},
		{BookingRequest{RequestID: "trip", Destination: "Paris", Nights: 0}, "failed",
			[]string{bookFlightName, bookHotelName, cancelFlightName}, []string{"flight"}},
		{BookingRequest{RequestID: "trip", Destination: "Nowhere", Nights: 3}, "failed",
			[]string{bookFlightName}, nil},
	} {
		t.Run(scenario.input.Destination+"-"+scenario.status, func(t *testing.T) {
			var calls []string
			result, err := executeSaga(scenario.input, invokeFixture(t, &calls))
			if err != nil {
				t.Fatal(err)
			}
			if result.Status != scenario.status || !reflect.DeepEqual(calls, scenario.wantCalls) {
				t.Fatalf("result = %+v, calls = %v; want %s, %v", result, calls, scenario.status, scenario.wantCalls)
			}
			var compensated []string
			for _, compensation := range result.Compensations {
				if compensation.Status != "cancelled" || compensation.Error != "" {
					t.Fatalf("failed compensation: %+v", compensation)
				}
				compensated = append(compensated, compensation.Service)
			}
			if !reflect.DeepEqual(compensated, scenario.wantCompensations) {
				t.Fatalf("compensations = %v, want %v", compensated, scenario.wantCompensations)
			}
			for _, booking := range result.Bookings {
				if booking.Confirmation != confirmation(booking.Service, "trip") ||
					booking.Destination != scenario.input.Destination {
					t.Fatalf("invalid booking identity: %+v", booking)
				}
			}
		})
	}
}

func TestCompensationFailureStillCancelsRemainingBookings(t *testing.T) {
	var calls []string
	result, err := executeSaga(BookingRequest{
		RequestID: "trip", Destination: "Tokyo", Nights: 3, SimulateCarFailure: true, SimulateCancellationFailure: "hotel",
	}, invokeFixture(t, &calls))
	var failure *compensationFailure
	if !errors.As(err, &failure) || failure.DurableTaskErrorType() != compensationErrorType ||
		err.Error() != "compensation incomplete after No rental cars available in Tokyo: hotel: simulated hotel cancellation outage" {
		t.Fatalf("unexpected compensation failure: %v", err)
	}
	want := []Cancellation{
		{Service: "hotel", Confirmation: "HT-trip", Status: "failed", Error: "simulated hotel cancellation outage"},
		{Service: "flight", Confirmation: "FL-trip", Status: "cancelled"},
	}
	if result.Status != "compensation_failed" || !reflect.DeepEqual(result.Compensations, want) {
		t.Fatalf("compensation status = %+v, want %+v", result, want)
	}
	wantCalls := []string{bookFlightName, bookHotelName, bookCarName, cancelHotelName, cancelFlightName}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", calls, wantCalls)
	}
}

func TestUnexpectedFailureIsNotAnExpectedRejection(t *testing.T) {
	var calls []string
	invoke := invokeFixture(t, &calls)
	unavailable := errors.New("hotel transport unavailable")
	result, err := executeSaga(BookingRequest{RequestID: "trip", Destination: "Paris", Nights: 5},
		func(name string, input, output any) error {
			if name == bookHotelName {
				return unavailable
			}
			return invoke(name, input, output)
		})
	if !errors.Is(err, unavailable) || result.Status != "failed" || len(result.Compensations) != 1 ||
		result.Compensations[0].Service != "flight" {
		t.Fatalf("unexpected failure was swallowed or left bookings uncompensated: %+v, %v", result, err)
	}
}

func TestCompensationRetriesAndEvidence(t *testing.T) {
	policy, err := compensationRetryPolicy().Normalized()
	if err != nil || policy.MaxAttempts != 3 || policy.InitialRetryInterval != 100*time.Millisecond ||
		policy.BackoffCoefficient != 2 || policy.RetryTimeout != 10*time.Second {
		t.Fatalf("unbounded/incorrect retry policy: %+v, %v", policy, err)
	}
	history := &api.OrchestrationHistory{}
	for _, name := range []string{
		bookFlightName, bookHotelName, bookCarName, cancelHotelName, cancelHotelName, cancelHotelName, cancelFlightName,
	} {
		history.Events = append(history.Events, &api.HistoryEvent{
			Type: api.HistoryEventTaskScheduled, TaskScheduled: &api.HistoryTaskScheduledEvent{Name: name},
		})
	}
	if attempts, err := verifyRetryHistory(history); err != nil || attempts != 3 {
		t.Fatalf("retry evidence = %d, %v", attempts, err)
	}
	history.Events = history.Events[:len(history.Events)-1]
	if _, err := verifyRetryHistory(history); err == nil {
		t.Fatal("history without the remaining flight compensation was accepted")
	}
}

func TestActivityValidationAndIdempotentFixture(t *testing.T) {
	input := activityInput(`{"request_id":"trip","destination":"Paris","nights":5}`)
	first, err := bookFlight(input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := bookFlight(input)
	if err != nil || first != second || first != (Booking{Service: "flight", Confirmation: "FL-trip", Destination: "Paris"}) {
		t.Fatalf("unstable fixture confirmation: %+v, %+v, %v", first, second, err)
	}
	for _, activity := range []task.Activity{bookFlight, bookHotel, bookCar, cancelFlight, cancelHotel, cancelCar} {
		if _, err := activity(activityInput(`{`)); err == nil {
			t.Fatal("malformed activity input accepted")
		}
	}
	if _, err := cancelFlight(activityInput(`{"booking":{"service":"hotel","confirmation":"HT-trip"}}`)); err == nil {
		t.Fatal("cancellation for the wrong provider was accepted")
	}
}
