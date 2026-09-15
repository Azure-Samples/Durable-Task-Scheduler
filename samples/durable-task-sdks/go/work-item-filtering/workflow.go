package main

import "github.com/microsoft/durabletask-go/task"

type numbers struct {
	A int `json:"a"`
	B int `json:"b"`
}

type greetingResult struct {
	Worker string `json:"worker"`
	Result string `json:"result"`
}

type mathResult struct {
	Worker string `json:"worker"`
	Result int    `json:"result"`
}

func greetingWorkflow(ctx *task.OrchestrationContext) (any, error) {
	var name string
	if err := ctx.GetInput(&name); err != nil {
		return nil, err
	}
	var result greetingResult
	if err := ctx.CallActivity(helloName, task.WithActivityInput(name)).Await(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func mathWorkflow(ctx *task.OrchestrationContext) (any, error) {
	var input numbers
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	var result mathResult
	if err := ctx.CallActivity(addName, task.WithActivityInput(input)).Await(&result); err != nil {
		return nil, err
	}
	return result, nil
}
