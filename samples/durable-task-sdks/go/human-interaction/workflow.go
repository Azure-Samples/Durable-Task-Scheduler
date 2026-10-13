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

	eventCtx, cancelEvent := ctx.WithCancel()
	timerCtx, cancelTimer := ctx.WithCancel()
	responseTask := eventCtx.WaitForSingleEvent(approvalEvent, -1)
	timer := timerCtx.CreateTimer(time.Duration(request.TimeoutSeconds) * time.Second)
	winner := ctx.WhenAny(responseTask, timer)

	var result ApprovalResult
	if winner == responseTask {
		var response ApprovalResponse
		responseErr := responseTask.Await(&response)
		cancelTimer()
		timerErr := timer.Await(nil)
		if responseErr != nil {
			return nil, fmt.Errorf("read approval response: %w", responseErr)
		}
		if timerErr != nil && !errors.Is(timerErr, task.ErrTaskCanceled) {
			return nil, fmt.Errorf("cancel approval timer: %w", timerErr)
		}
		if err := ctx.CallActivity(processName, task.WithActivityInput(ProcessInput{
			RequestID: request.RequestID, Response: response,
		})).Await(&result); err != nil {
			return nil, fmt.Errorf("process approval: %w", err)
		}
	} else {
		timerErr := timer.Await(nil)
		cancelEvent()
		responseErr := responseTask.Await(nil)
		if timerErr != nil {
			return nil, fmt.Errorf("approval timer: %w", timerErr)
		}
		if responseErr != nil && !errors.Is(responseErr, task.ErrTaskCanceled) {
			return nil, fmt.Errorf("cancel approval wait: %w", responseErr)
		}
		var err error
		result, err = approvalOutcome(request.RequestID, nil)
		if err != nil {
			return nil, err
		}
	}
	if err := ctx.SetCustomStatusValue(result); err != nil {
		return nil, err
	}
	return result, nil
}
