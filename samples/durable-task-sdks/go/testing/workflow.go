package main

import "github.com/microsoft/durabletask-go/task"

const orderWorkflowName = "GoTestingOrder"

type item struct {
	Name           string `json:"name"`
	Quantity       int64  `json:"quantity"`
	UnitPriceCents int64  `json:"unitPriceCents"`
}

type order struct {
	Customer string `json:"customer"`
	Items    []item `json:"items"`
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
	return orderResult{
		PaymentID: payment, TrackingID: tracking,
		TotalCents: total, Status: "completed",
	}, nil
}

func orderWorkflow(ctx *task.OrchestrationContext) (any, error) {
	var input order
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	return processOrder(input, durableSteps{ctx: ctx})
}

// The workflow uses durable activities in the app and local steps in unit tests.
type durableSteps struct {
	ctx *task.OrchestrationContext
}

func (s durableSteps) Validate(input order) error {
	return s.ctx.CallActivity(validateName, task.WithActivityInput(input)).Await(nil)
}

func (s durableSteps) Charge(amount int64) (string, error) {
	var result string
	err := s.ctx.CallActivity(chargeName, task.WithActivityInput(amount)).Await(&result)
	return result, err
}

func (s durableSteps) Ship(input shipment) (string, error) {
	var result string
	err := s.ctx.CallActivity(shipName, task.WithActivityInput(input)).Await(&result)
	return result, err
}
