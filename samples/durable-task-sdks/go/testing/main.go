package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
	"github.com/microsoft/durabletask-go/task"
)

type item struct {
	Name           string `json:"name"`
	Quantity       int64  `json:"quantity"`
	UnitPriceCents int64  `json:"unitPriceCents"`
}

type order struct {
	Customer string `json:"customer"`
	Items    []item `json:"items"`
}

type shipment struct {
	Customer  string `json:"customer"`
	ItemCount int    `json:"itemCount"`
}

type orderResult struct {
	PaymentID  string `json:"paymentId"`
	TrackingID string `json:"trackingId"`
	TotalCents int64  `json:"totalCents"`
	Status     string `json:"status"`
}

type orderSteps interface {
	Validate(order) error
	Charge(int64) (string, error)
	Ship(shipment) (string, error)
}

func processOrder(input order, steps orderSteps) (orderResult, error) {
	if err := steps.Validate(input); err != nil {
		return orderResult{}, err
	}
	total, err := totalCents(input.Items)
	if err != nil {
		return orderResult{}, err
	}
	payment, err := steps.Charge(total)
	if err != nil {
		return orderResult{}, err
	}
	tracking, err := steps.Ship(shipment{Customer: input.Customer, ItemCount: len(input.Items)})
	if err != nil {
		return orderResult{}, err
	}
	return orderResult{payment, tracking, total, "completed"}, nil
}

func validateOrder(input order) error {
	if strings.TrimSpace(input.Customer) == "" {
		return errors.New("order must have a customer name")
	}
	if len(input.Items) == 0 {
		return errors.New("order must contain at least one item")
	}
	_, err := totalCents(input.Items)
	return err
}

func totalCents(items []item) (int64, error) {
	var total int64
	for _, item := range items {
		if item.Quantity <= 0 || item.UnitPriceCents <= 0 {
			return 0, fmt.Errorf("invalid quantity or price for %q", item.Name)
		}
		if item.Quantity > math.MaxInt64/item.UnitPriceCents {
			return 0, errors.New("line total exceeds supported amount")
		}
		line := item.Quantity * item.UnitPriceCents
		if total > math.MaxInt64-line {
			return 0, errors.New("order total exceeds supported amount")
		}
		total += line
	}
	return total, nil
}

func chargePayment(amount int64) (string, error) {
	if amount <= 0 {
		return "", errors.New("payment amount must be positive")
	}
	// A deterministic stand-in for an idempotent payment gateway.
	return fmt.Sprintf("PAY-%d", amount), nil
}

func shipOrder(input shipment) (string, error) {
	if input.Customer == "" || input.ItemCount <= 0 {
		return "", errors.New("shipment requires a customer and items")
	}
	return fmt.Sprintf("TRACK-%s-%d", strings.ToUpper(input.Customer), input.ItemCount), nil
}

type durableSteps struct {
	ctx *task.OrchestrationContext
}

func (s durableSteps) Validate(input order) error {
	return s.ctx.CallActivity("GoTestingValidate", task.WithActivityInput(input)).Await(nil)
}

func (s durableSteps) Charge(amount int64) (string, error) {
	var result string
	err := s.ctx.CallActivity("GoTestingCharge", task.WithActivityInput(amount)).Await(&result)
	return result, err
}

func (s durableSteps) Ship(input shipment) (string, error) {
	var result string
	err := s.ctx.CallActivity("GoTestingShip", task.WithActivityInput(input)).Await(&result)
	return result, err
}

func orderWorkflow(ctx *task.OrchestrationContext) (any, error) {
	var input order
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	return processOrder(input, durableSteps{ctx})
}

func validateActivity(ctx task.ActivityContext) (any, error) {
	var input order
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	return nil, validateOrder(input)
}

func chargeActivity(ctx task.ActivityContext) (any, error) {
	var amount int64
	if err := ctx.GetInput(&amount); err != nil {
		return nil, err
	}
	return chargePayment(amount)
}

func shipActivity(ctx task.ActivityContext) (any, error) {
	var input shipment
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	return shipOrder(input)
}

func registry() (*task.TaskRegistry, error) {
	r := task.NewTaskRegistry()
	err := errors.Join(
		r.AddOrchestratorN("GoTestingOrder", orderWorkflow),
		r.AddActivityN("GoTestingValidate", validateActivity),
		r.AddActivityN("GoTestingCharge", chargeActivity),
		r.AddActivityN("GoTestingShip", shipActivity),
	)
	return r, err
}

func run(ctx context.Context) error {
	r, err := registry()
	if err != nil {
		return err
	}
	return sample.WithHost(ctx, r, func(ctx context.Context, c *dts.Client) error {
		cases := []struct {
			name  string
			input order
			want  orderResult
			cause string
		}{
			{"single", order{"Alice", []item{{"Widget", 2, 1000}}}, orderResult{"PAY-2000", "TRACK-ALICE-1", 2000, "completed"}, ""},
			{"multiple", order{"Bob", []item{{"Widget", 3, 2500}, {"Gadget", 1, 9999}}}, orderResult{"PAY-17499", "TRACK-BOB-2", 17499, "completed"}, ""},
			{"missing-customer", order{"", []item{{"Widget", 1, 1000}}}, orderResult{}, "customer name"},
			{"empty", order{"Eve", nil}, orderResult{}, "at least one item"},
			{"invalid-quantity", order{"Mallory", []item{{"Widget", 0, 1000}}}, orderResult{}, "invalid quantity"},
		}
		for _, test := range cases {
			id, err := c.ScheduleNewOrchestration(ctx, "GoTestingOrder",
				api.WithInstanceID(sample.ID("testing-"+test.name)), api.WithInput(test.input))
			if err != nil {
				return err
			}
			if test.cause == "" {
				var result orderResult
				if err := sample.Wait(ctx, c, id, &result); err != nil {
					return err
				}
				if result != test.want {
					return fmt.Errorf("%s: got %+v, want %+v", test.name, result, test.want)
				}
			} else {
				metadata, err := c.WaitForOrchestrationCompletion(ctx, id)
				if err != nil {
					return err
				}
				if metadata.RuntimeStatus != api.RUNTIME_STATUS_FAILED || !hasCause(metadata.FailureDetails, test.cause) {
					return fmt.Errorf("%s: expected failure containing %q, got %s: %v", test.name, test.cause, metadata.RuntimeStatus, metadata.FailureDetails)
				}
			}
			fmt.Printf("Verified %s: %s\n", test.name, id)
		}
		return nil
	})
}

func hasCause(details *api.FailureDetails, text string) bool {
	for current := details; current != nil; current = current.InnerFailure {
		if strings.Contains(current.ErrorMessage, text) {
			return true
		}
	}
	return false
}

func main() {
	sample.Main("testing", run)
}
