package main

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/testutil"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
	"github.com/microsoft/durabletask-go/task"
)

func integrationOrders(task.ActivityContext) (any, error) {
	return []Order{
		{ID: "order-1"},
		{ID: "order-2", FailAt: "inventory"},
		{ID: "order-3", FailAt: "payment"},
		{ID: "order-4", FailAt: "shipping"},
		{ID: "order-5", FailAt: "notification"},
	}, nil
}

func TestIntegration(t *testing.T) {
	ctx := testutil.IntegrationContext(t)
	// Only the source fixture changes; the parent, children, and business activities are real.
	r := task.NewTaskRegistry()
	err := errors.Join(
		r.AddOrchestratorN(orchestrationName, ordersOrchestration),
		r.AddOrchestratorN(orderName, processOrderOrchestration),
		r.AddActivityN(getOrdersName, integrationOrders),
		r.AddActivityN(inventoryName, checkInventory),
		r.AddActivityN(paymentName, chargePayment),
		r.AddActivityN(shippingName, shipOrder),
		r.AddActivityN(notificationName, notifyCustomer),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = sample.WithHost(ctx, r, func(ctx context.Context, c *dts.Client) (err error) {
		id, err := c.ScheduleNewOrchestration(ctx, orchestrationName,
			api.WithInstanceID(sample.ID("sub-orchestrations-test")))
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
		return testutil.Require(reflect.DeepEqual(result, want), "order summary = %+v, want %+v", result, want)
	})
	if err != nil {
		t.Fatal(err)
	}
}
