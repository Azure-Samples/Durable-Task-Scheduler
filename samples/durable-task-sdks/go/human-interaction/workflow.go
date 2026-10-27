package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/microsoft/durabletask-go/task"
)

const (
	orchestrationName = "GoHumanInteraction"
	submitName        = "GoHumanInteractionSubmitApprovalRequest"
	processName       = "GoHumanInteractionProcessApproval"
	approvalEvent     = "GoHumanInteractionApprovalResponse"
)

func humanInteraction(ctx *task.OrchestrationContext) (any, error) {
	var request ApprovalRequest
	if err := ctx.GetInput(&request); err != nil {
		return nil, err
	}
	if err := request.validate(); err != nil {
		return nil, err
	}
	var submission ApprovalResult
	if err := ctx.CallActivity(submitName, task.WithActivityInput(request)).Await(&submission); err != nil {
		return nil, fmt.Errorf("submit approval: %w", err)
	}
	if err := ctx.SetCustomStatusValue(submission); err != nil {
		return nil, err
	}

	var response ApprovalResponse
	err := ctx.WaitForSingleEvent(
		approvalEvent, time.Duration(request.TimeoutSeconds)*time.Second,
	).Await(&response)
	var result ApprovalResult
	if errors.Is(err, task.ErrTaskCanceled) {
		result, err = approvalOutcome(request.RequestID, nil)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, fmt.Errorf("read approval response: %w", err)
	} else if err := ctx.CallActivity(processName, task.WithActivityInput(ProcessInput{
		RequestID: request.RequestID, Response: response,
	})).Await(&result); err != nil {
		return nil, fmt.Errorf("process approval: %w", err)
	}
	if err := ctx.SetCustomStatusValue(result); err != nil {
		return nil, err
	}
	return result, nil
}
