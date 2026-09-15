package main

import (
	"errors"
	"fmt"

	"github.com/microsoft/durabletask-go/task"
)

type Order struct {
	ID     string `json:"id"`
	FailAt string `json:"simulate_failure_at,omitempty"`
}

func (order Order) validate() error {
	if order.ID == "" {
		return errors.New("order ID must not be empty")
	}
	switch order.FailAt {
	case "", "inventory", "payment", "shipping", "notification":
		return nil
	default:
		return fmt.Errorf("unknown simulated failure step %q", order.FailAt)
	}
}

func getOrders(task.ActivityContext) (any, error) {
	// Simulation only: a small batch of orders ready for fulfillment.
	return []Order{{ID: "order-1"}, {ID: "order-2"}}, nil
}

func simulateStep(ctx task.ActivityContext, step string) (any, error) {
	var order Order
	if err := ctx.GetInput(&order); err != nil {
		return nil, err
	}
	if err := order.validate(); err != nil {
		return nil, err
	}
	// Simulation only: no inventory, payment, shipping, or notification service is called.
	return order.FailAt != step, nil
}

func checkInventory(ctx task.ActivityContext) (any, error) { return simulateStep(ctx, "inventory") }
func chargePayment(ctx task.ActivityContext) (any, error)  { return simulateStep(ctx, "payment") }
func shipOrder(ctx task.ActivityContext) (any, error)      { return simulateStep(ctx, "shipping") }
func notifyCustomer(ctx task.ActivityContext) (any, error) { return simulateStep(ctx, "notification") }
