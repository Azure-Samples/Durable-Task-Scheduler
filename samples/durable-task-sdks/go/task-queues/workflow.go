package main

import (
	"fmt"
	"strings"

	"github.com/microsoft/durabletask-go/task"
)

const queueTag = "DTS_TaskQueue"

type order struct {
	ID string `json:"id"`
}

func (o order) validate() error {
	if strings.TrimSpace(o.ID) == "" {
		return fmt.Errorf("order ID must not be empty")
	}
	return nil
}

type orderResult struct {
	OrderID      string `json:"order_id"`
	Accepted     string `json:"accepted"`
	Partner      string `json:"partner"`
	ChildPartner string `json:"child_partner"`
}

func processOrder(ctx *task.OrchestrationContext) (any, error) {
	var input order
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	if err := input.validate(); err != nil {
		return nil, err
	}
	result := orderResult{OrderID: input.ID}
	// No tag override: this activity inherits the parent's orders queue.
	if err := ctx.CallActivity(acceptName,
		task.WithActivityInput(input)).Await(&result.Accepted); err != nil {
		return nil, err
	}
	if err := ctx.CallActivity(partnerName,
		task.WithActivityInput(input),
		task.WithActivityTags(map[string]string{queueTag: "partner-api"}),
	).Await(&result.Partner); err != nil {
		return nil, err
	}
	if err := ctx.CallSubOrchestrator(childName,
		task.WithSubOrchestratorInput(input),
		task.WithSubOrchestrationInstanceID(string(ctx.ID)+"-partner"),
		task.WithSubOrchestrationTags(map[string]string{queueTag: "partner-api"}),
	).Await(&result.ChildPartner); err != nil {
		return nil, err
	}
	return result, nil
}

func partnerOrder(ctx *task.OrchestrationContext) (any, error) {
	var input order
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	if err := input.validate(); err != nil {
		return nil, err
	}
	var result string
	// This activity inherits partner-api from the child orchestration.
	if err := ctx.CallActivity(partnerName,
		task.WithActivityInput(input)).Await(&result); err != nil {
		return nil, err
	}
	return result, nil
}
