package main

import "github.com/microsoft/durabletask-go/task"

type payloadWorkflow struct {
	orchestrator string
	echo         string
	process      string
}

func newPayloadWorkflow(runID string) payloadWorkflow {
	return payloadWorkflow{
		orchestrator: "GoLargePayloadOrchestrator-" + runID,
		echo:         "GoLargePayloadEchoData-" + runID,
		process:      "GoLargePayloadProcessData-" + runID,
	}
}

func (w payloadWorkflow) orchestrate(ctx *task.OrchestrationContext) (any, error) {
	var content string
	if err := ctx.GetInput(&content); err != nil {
		return nil, err
	}
	var echoed string
	if err := ctx.CallActivity(w.echo, task.WithActivityInput(content)).Await(&echoed); err != nil {
		return nil, err
	}
	var result payloadResult
	if err := ctx.CallActivity(w.process, task.WithActivityInput(echoed)).Await(&result); err != nil {
		return nil, err
	}
	return result, nil
}
