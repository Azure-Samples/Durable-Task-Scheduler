package main

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
	"github.com/microsoft/durabletask-go/task"
)

const (
	orchestrationName                   = "GoSagaTravelBooking"
	bookFlightName                      = "GoSagaBookFlight"
	bookHotelName                       = "GoSagaBookHotel"
	bookCarName                         = "GoSagaBookCar"
	cancelFlightName                    = "GoSagaCancelFlight"
	cancelHotelName                     = "GoSagaCancelHotel"
	cancelCarName                       = "GoSagaCancelCar"
	bookingRejectedType   api.ErrorType = "GoSagaBookingRejected"
	cancellationErrorType api.ErrorType = "GoSagaCancellationUnavailable"
	compensationErrorType api.ErrorType = "GoSagaCompensationFailed"
)

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

type SagaResult struct {
	Status        string         `json:"status"`
	Destination   string         `json:"destination"`
	Bookings      []Booking      `json:"bookings,omitempty"`
	Error         string         `json:"error,omitempty"`
	Compensations []Cancellation `json:"compensations,omitempty"`
}

type bookingRejected struct{ message string }

func (err *bookingRejected) Error() string                   { return err.message }
func (*bookingRejected) DurableTaskErrorType() api.ErrorType { return bookingRejectedType }
func (*bookingRejected) NonRetriable() bool                  { return true }

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

func bookFlight(ctx task.ActivityContext) (any, error)   { return makeBooking(ctx, "flight") }
func bookHotel(ctx task.ActivityContext) (any, error)    { return makeBooking(ctx, "hotel") }
func bookCar(ctx task.ActivityContext) (any, error)      { return makeBooking(ctx, "car") }
func cancelFlight(ctx task.ActivityContext) (any, error) { return cancelBooking(ctx, "flight") }
func cancelHotel(ctx task.ActivityContext) (any, error)  { return cancelBooking(ctx, "hotel") }
func cancelCar(ctx task.ActivityContext) (any, error)    { return cancelBooking(ctx, "car") }

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

func compensationRetryPolicy() *task.RetryPolicy {
	return &task.RetryPolicy{
		MaxAttempts: 3, InitialRetryInterval: 100 * time.Millisecond, BackoffCoefficient: 2,
		MaxRetryInterval: 500 * time.Millisecond, RetryTimeout: 10 * time.Second,
	}
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

func newRegistry() (*task.TaskRegistry, error) {
	r := task.NewTaskRegistry()
	return r, errors.Join(
		r.AddOrchestratorN(orchestrationName, travelBookingSaga),
		r.AddActivityN(bookFlightName, bookFlight),
		r.AddActivityN(bookHotelName, bookHotel),
		r.AddActivityN(bookCarName, bookCar),
		r.AddActivityN(cancelFlightName, cancelFlight),
		r.AddActivityN(cancelHotelName, cancelHotel),
		r.AddActivityN(cancelCarName, cancelCar),
	)
}

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
	if err := sample.Require(reflect.DeepEqual(result, want), "%s result = %+v, want %+v", scenario.Name, result, want); err != nil {
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

func run(ctx context.Context) error {
	r, err := newRegistry()
	if err != nil {
		return err
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
	return sample.WithHost(ctx, r, func(ctx context.Context, c *dts.Client) error {
		for _, scenario := range scenarios {
			if err := verifyScenario(ctx, c, scenario); err != nil {
				return err
			}
		}
		return nil
	})
}

func stopOnError(c *dts.Client, id api.InstanceID, runErr *error) {
	if *runErr == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	state, err := c.FetchOrchestrationMetadata(ctx, id)
	if err == nil && !state.IsComplete() {
		err = c.TerminateOrchestration(ctx, id)
		if err == nil {
			_, err = c.WaitForOrchestrationCompletion(ctx, id)
		}
	}
	*runErr = errors.Join(*runErr, err)
}

func main() {
	sample.Main("saga", run)
}
