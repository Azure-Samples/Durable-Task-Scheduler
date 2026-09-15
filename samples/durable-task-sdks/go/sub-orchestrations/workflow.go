package main

import (
	"errors"
	"fmt"

	"github.com/microsoft/durabletask-go/task"
)

const (
	orchestrationName = "GoSubOrchestrationsOrders"
	orderName         = "GoSubOrchestrationsProcessOrder"
	getOrdersName     = "GoSubOrchestrationsGetOrders"
	inventoryName     = "GoSubOrchestrationsCheckAndUpdateInventory"
	paymentName       = "GoSubOrchestrationsChargePayment"
	shippingName      = "GoSubOrchestrationsShipOrder"
	notificationName  = "GoSubOrchestrationsNotifyCustomer"
)

type OrderResult struct {
	Order  string   `json:"order"`
	Status string   `json:"status"`
	Reason string   `json:"reason,omitempty"`
	Steps  []string `json:"steps"`
}

type OrderSummary struct {
	Orders         []string      `json:"orders"`
	Results        []OrderResult `json:"results"`
	TotalCompleted int           `json:"total_completed"`
	TotalFailed    int           `json:"total_failed"`
}

func processOrder(order Order, call func(string, Order) (bool, error)) (OrderResult, error) {
	if err := order.validate(); err != nil {
		return OrderResult{}, err
	}
	steps := []struct {
		name   string
		phase  string
		reason string
	}{
		{inventoryName, "inventory", "out of stock"},
		{paymentName, "payment", "payment failed"},
		{shippingName, "shipping", "shipping failed"},
		{notificationName, "notification", "customer notification failed"},
	}
	result := OrderResult{Order: order.ID, Status: "completed", Steps: []string{}}
	for _, step := range steps {
		result.Steps = append(result.Steps, step.phase)
		ok, err := call(step.name, order)
		if err != nil {
			return OrderResult{}, fmt.Errorf("order %s, %s activity: %w", order.ID, step.phase, err)
		}
		if !ok {
			result.Status = "failed"
			result.Reason = step.reason
			return result, nil
		}
	}
	return result, nil
}

func processOrderOrchestration(ctx *task.OrchestrationContext) (any, error) {
	var order Order
	if err := ctx.GetInput(&order); err != nil {
		return nil, err
	}
	return processOrder(order, func(name string, input Order) (bool, error) {
		var ok bool
		err := ctx.CallActivity(name, task.WithActivityInput(input)).Await(&ok)
		return ok, err
	})
}

func ordersOrchestration(ctx *task.OrchestrationContext) (any, error) {
	var orders []Order
	if err := ctx.CallActivity(getOrdersName).Await(&orders); err != nil {
		return nil, fmt.Errorf("get orders: %w", err)
	}
	if len(orders) > 100 {
		return nil, errors.New("order batch exceeds the sample limit of 100")
	}
	seen := make(map[string]bool, len(orders))
	for _, order := range orders {
		if err := order.validate(); err != nil {
			return nil, err
		}
		if seen[order.ID] {
			return nil, fmt.Errorf("duplicate order ID %q", order.ID)
		}
		seen[order.ID] = true
	}

	pending := make([]task.Task, len(orders))
	for i, order := range orders {
		pending[i] = ctx.CallSubOrchestrator(orderName,
			task.WithSubOrchestrationInstanceID(string(ctx.ID)+"-"+order.ID),
			task.WithSubOrchestratorInput(order))
	}
	if err := ctx.WhenAll(pending...); err != nil {
		return nil, fmt.Errorf("process child orders: %w", err)
	}
	summary := OrderSummary{Orders: make([]string, len(orders)), Results: make([]OrderResult, len(orders))}
	for i, child := range pending {
		result := &summary.Results[i]
		if err := child.Await(result); err != nil {
			return nil, fmt.Errorf("decode order %s: %w", orders[i].ID, err)
		}
		if result.Order != orders[i].ID {
			return nil, fmt.Errorf("child returned the wrong order: %+v", result)
		}
		summary.Orders[i] = result.Order
		switch result.Status {
		case "completed":
			summary.TotalCompleted++
		case "failed":
			summary.TotalFailed++
		default:
			return nil, fmt.Errorf("unexpected order result: %+v", result)
		}
	}
	return summary, nil
}
