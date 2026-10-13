package main

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/testutil"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
)

type Scenario struct {
	Name        string
	Input       BookingRequest
	Status      string
	Error       string
	Booked      []string
	Compensated []string
}

func expectedResult(scenario Scenario) SagaResult {
	result := SagaResult{Status: scenario.Status, Destination: scenario.Input.Destination, Error: scenario.Error}
	prefixes := map[string]string{"flight": "FL-", "hotel": "HT-", "car": "CR-"}
	for _, service := range scenario.Booked {
		result.Bookings = append(result.Bookings, Booking{
			Service: service, Destination: scenario.Input.Destination, Confirmation: prefixes[service] + scenario.Input.RequestID,
		})
	}
	for _, service := range scenario.Compensated {
		cancellation := Cancellation{
			Service: service, Confirmation: prefixes[service] + scenario.Input.RequestID, Status: "cancelled",
		}
		if service == scenario.Input.SimulateCancellationFailure {
			cancellation.Status = "failed"
			cancellation.Error = "simulated " + service + " cancellation outage"
		}
		result.Compensations = append(result.Compensations, cancellation)
	}
	return result
}

func verifyRetryHistory(history *api.OrchestrationHistory) (int, error) {
	if history == nil {
		return 0, errors.New("missing compensation history")
	}
	counts := make(map[string]int)
	for _, event := range history.Events {
		if event == nil {
			return 0, errors.New("nil event in compensation history")
		}
		if event.Type == api.HistoryEventTaskScheduled && event.TaskScheduled != nil {
			counts[event.TaskScheduled.Name]++
		}
	}
	want := map[string]int{
		bookFlightName: 1, bookHotelName: 1, bookCarName: 1, cancelHotelName: 3, cancelFlightName: 1,
	}
	if !reflect.DeepEqual(counts, want) {
		return 0, fmt.Errorf("compensation activity attempts = %v, want %v", counts, want)
	}
	return counts[cancelHotelName], nil
}

func verifyScenario(ctx context.Context, c *dts.Client, scenario Scenario) (err error) {
	id := sample.ID("saga-" + scenario.Name)
	scenario.Input.RequestID = string(id)
	if _, err := c.ScheduleNewOrchestration(ctx, orchestrationName,
		api.WithInstanceID(id), api.WithInput(scenario.Input)); err != nil {
		return err
	}
	defer stopOnError(c, id, &err)

	want := expectedResult(scenario)
	var result SagaResult
	runtimeStatus := api.RUNTIME_STATUS_COMPLETED
	retryAttempts := 0
	if scenario.Status == "compensation_failed" {
		metadata, err := c.WaitForOrchestrationCompletion(ctx, id, api.WithFetchPayloads(true))
		if err != nil {
			return err
		}
		runtimeStatus = metadata.RuntimeStatus
		const wantFailure = "compensation incomplete after No rental cars available in Tokyo: hotel: simulated hotel cancellation outage"
		if metadata.RuntimeStatus != api.RUNTIME_STATUS_FAILED || metadata.FailureDetails == nil ||
			metadata.FailureDetails.ErrorType != compensationErrorType || metadata.FailureDetails.ErrorMessage != wantFailure {
			return fmt.Errorf("expected explicit compensation failure, got %s: %+v", metadata.RuntimeStatus, metadata.FailureDetails)
		}
		if err := metadata.ReadCustomStatus(&result); err != nil {
			return err
		}
		history, err := c.GetOrchestrationHistory(ctx, id, api.HistoryQuery{MaxEvents: 200})
		if err != nil {
			return fmt.Errorf("verify exhausted compensation retries: %w", err)
		}
		retryAttempts, err = verifyRetryHistory(history)
		if err != nil {
			return err
		}
	} else if err := sample.Wait(ctx, c, id, &result); err != nil {
		return err
	}
	if err := testutil.Require(reflect.DeepEqual(result, want), "%s result = %+v, want %+v", scenario.Name, result, want); err != nil {
		return err
	}
	return sample.PrintJSON(struct {
		Scenario                  string         `json:"scenario"`
		InstanceID                api.InstanceID `json:"instance_id"`
		RuntimeStatus             string         `json:"runtime_status"`
		HotelCancellationAttempts int            `json:"hotel_cancellation_attempts,omitempty"`
		Result                    SagaResult     `json:"result"`
	}{scenario.Name, id, runtimeStatus.String(), retryAttempts, result})
}

func TestIntegration(t *testing.T) {
	ctx := testutil.IntegrationContext(t)
	r, err := newRegistry()
	if err != nil {
		t.Fatal(err)
	}
	scenarios := []Scenario{
		{Name: "success", Input: BookingRequest{Destination: "Paris", Nights: 5},
			Status: "success", Booked: []string{"flight", "hotel", "car"}},
		{Name: "car-failure", Input: BookingRequest{Destination: "Tokyo", Nights: 3, SimulateCarFailure: true},
			Status: "failed", Error: "No rental cars available in Tokyo",
			Booked: []string{"flight", "hotel"}, Compensated: []string{"hotel", "flight"}},
		{Name: "hotel-failure", Input: BookingRequest{Destination: "Paris", Nights: 0},
			Status: "failed", Error: "Invalid hotel booking: 0 nights",
			Booked: []string{"flight"}, Compensated: []string{"flight"}},
		{Name: "flight-failure", Input: BookingRequest{Destination: "Nowhere", Nights: 3},
			Status: "failed", Error: "No flights available to Nowhere"},
		{Name: "compensation-failure", Input: BookingRequest{
			Destination: "Tokyo", Nights: 3, SimulateCarFailure: true, SimulateCancellationFailure: "hotel",
		}, Status: "compensation_failed", Error: "No rental cars available in Tokyo",
			Booked: []string{"flight", "hotel"}, Compensated: []string{"hotel", "flight"}},
	}
	err = sample.WithHost(ctx, r, func(ctx context.Context, c *dts.Client) error {
		for _, scenario := range scenarios {
			if err := verifyScenario(ctx, c, scenario); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
