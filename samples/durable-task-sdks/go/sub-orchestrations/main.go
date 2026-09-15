package main

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
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

func getOrders(task.ActivityContext) (any, error) {
	return []Order{
		{ID: "order-1"},
		{ID: "order-2", FailAt: "inventory"},
		{ID: "order-3", FailAt: "payment"},
		{ID: "order-4", FailAt: "shipping"},
		{ID: "order-5", FailAt: "notification"},
	}, nil
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

func newRegistry() (*task.TaskRegistry, error) {
	r := task.NewTaskRegistry()
	return r, errors.Join(
		r.AddOrchestratorN(orchestrationName, ordersOrchestration),
		r.AddOrchestratorN(orderName, processOrderOrchestration),
		r.AddActivityN(getOrdersName, getOrders),
		r.AddActivityN(inventoryName, checkInventory),
		r.AddActivityN(paymentName, chargePayment),
		r.AddActivityN(shippingName, shipOrder),
		r.AddActivityN(notificationName, notifyCustomer),
	)
}

func run(ctx context.Context) error {
	r, err := newRegistry()
	if err != nil {
		return err
	}
	return sample.WithHost(ctx, r, func(ctx context.Context, c *dts.Client) (err error) {
		id, err := c.ScheduleNewOrchestration(ctx, orchestrationName,
			api.WithInstanceID(sample.ID("sub-orchestrations")))
		if err != nil {
			return err
		}
		defer stopOnError(c, id, &err)

		var result OrderSummary
		if err := sample.Wait(ctx, c, id, &result); err != nil {
			return err
		}
		want := OrderSummary{
			Orders: []string{"order-1", "order-2", "order-3", "order-4", "order-5"},
			Results: []OrderResult{
				{Order: "order-1", Status: "completed", Steps: []string{"inventory", "payment", "shipping", "notification"}},
				{Order: "order-2", Status: "failed", Reason: "out of stock", Steps: []string{"inventory"}},
				{Order: "order-3", Status: "failed", Reason: "payment failed", Steps: []string{"inventory", "payment"}},
				{Order: "order-4", Status: "failed", Reason: "shipping failed", Steps: []string{"inventory", "payment", "shipping"}},
				{Order: "order-5", Status: "failed", Reason: "customer notification failed", Steps: []string{"inventory", "payment", "shipping", "notification"}},
			},
			TotalCompleted: 1, TotalFailed: 4,
		}
		if err := sample.Require(reflect.DeepEqual(result, want), "order summary = %+v, want %+v", result, want); err != nil {
			return err
		}
		return sample.PrintJSON(struct {
			InstanceID api.InstanceID `json:"instance_id"`
			Summary    OrderSummary   `json:"summary"`
		}{id, result})
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
	sample.Main("sub-orchestrations", run)
}
