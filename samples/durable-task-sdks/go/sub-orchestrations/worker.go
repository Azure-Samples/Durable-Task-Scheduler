package main

import (
	"errors"

	"github.com/microsoft/durabletask-go/task"
)

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
