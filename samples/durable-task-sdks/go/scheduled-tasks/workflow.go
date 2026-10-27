package main

import "github.com/microsoft/durabletask-go/task"

type reportInput struct {
	ScheduleID string `json:"schedule_id"`
	Phase      string `json:"phase"`
	Region     string `json:"region"`
}

type reportResult struct {
	ScheduleID string `json:"schedule_id"`
	Phase      string `json:"phase"`
	Message    string `json:"message"`
}

func reportWorkflow(ctx *task.OrchestrationContext) (any, error) {
	var input reportInput
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	var result reportResult
	if err := ctx.CallActivity(activityName, task.WithActivityInput(input)).Await(&result); err != nil {
		return nil, err
	}
	return result, nil
}
