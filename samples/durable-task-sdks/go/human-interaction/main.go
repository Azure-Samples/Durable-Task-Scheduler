package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
	"github.com/microsoft/durabletask-go/task"
)

const (
	orchestrationName = "GoHumanInteraction"
	submitName        = "GoHumanInteractionSubmitApprovalRequest"
	processName       = "GoHumanInteractionProcessApproval"
	approvalEvent     = "GoHumanInteractionApprovalResponse"
)

type ApprovalRequest struct {
	RequestID      string `json:"request_id"`
	Requester      string `json:"requester"`
	Item           string `json:"item"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

func (request ApprovalRequest) validate() error {
	if strings.TrimSpace(request.RequestID) == "" || strings.TrimSpace(request.Requester) == "" ||
		strings.TrimSpace(request.Item) == "" {
		return errors.New("approval requires a request ID, requester, and item")
	}
	if request.TimeoutSeconds <= 0 || request.TimeoutSeconds > 24*60*60 {
		return errors.New("approval timeout must be between one second and 24 hours")
	}
	return nil
}

type ApprovalResponse struct {
	IsApproved *bool  `json:"is_approved"`
	Approver   string `json:"approver"`
	Comments   string `json:"comments"`
}

type ApprovalResult struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
	Approver  string `json:"approver,omitempty"`
}

type ProcessInput struct {
	RequestID string           `json:"request_id"`
	Response  ApprovalResponse `json:"response"`
}

func submitApprovalRequest(ctx task.ActivityContext) (any, error) {
	var request ApprovalRequest
	if err := ctx.GetInput(&request); err != nil {
		return nil, err
	}
	if err := request.validate(); err != nil {
		return nil, err
	}
	// Simulation only: a real activity would idempotently notify an approver.
	return ApprovalResult{RequestID: request.RequestID, Status: "Pending"}, nil
}

func approvalOutcome(requestID string, response *ApprovalResponse) (ApprovalResult, error) {
	if strings.TrimSpace(requestID) == "" {
		return ApprovalResult{}, errors.New("request ID must not be empty")
	}
	if response == nil {
		return ApprovalResult{RequestID: requestID, Status: "Timeout"}, nil
	}
	if response.IsApproved == nil || strings.TrimSpace(response.Approver) == "" {
		return ApprovalResult{}, errors.New("response requires an explicit decision and approver")
	}
	status := "Rejected"
	if *response.IsApproved {
		status = "Approved"
	}
	return ApprovalResult{RequestID: requestID, Status: status, Approver: response.Approver}, nil
}

func processApproval(ctx task.ActivityContext) (any, error) {
	var input ProcessInput
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	// Simulation only: no database is updated by this sample.
	return approvalOutcome(input.RequestID, &input.Response)
}

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

func newRegistry() (*task.TaskRegistry, error) {
	r := task.NewTaskRegistry()
	return r, errors.Join(
		r.AddOrchestratorN(orchestrationName, humanInteraction),
		r.AddActivityN(submitName, submitApprovalRequest),
		r.AddActivityN(processName, processApproval),
	)
}

func verifyRequest(ctx context.Context, c *dts.Client, scenario string, decision *bool) (err error) {
	id := sample.ID("human-interaction-" + scenario)
	request := ApprovalRequest{
		RequestID: string(id), Requester: "Console User", Item: "Vacation Request", TimeoutSeconds: 10,
	}
	if decision == nil {
		request.TimeoutSeconds = 1
	}
	if _, err := c.ScheduleNewOrchestration(ctx, orchestrationName,
		api.WithInstanceID(id), api.WithInput(request)); err != nil {
		return err
	}
	defer stopOnError(c, id, &err)

	want := ApprovalResult{RequestID: string(id), Status: "Timeout"}
	if decision != nil {
		if err := sample.Until(ctx, 50*time.Millisecond, func() (bool, error) {
			state, err := c.FetchOrchestrationMetadata(ctx, id, api.WithFetchPayloads(true))
			if err != nil {
				return false, err
			}
			if state.IsComplete() {
				return false, fmt.Errorf("%s ended before a response: %s", id, state.RuntimeStatus)
			}
			if state.SerializedCustomStatus == "" {
				return false, nil
			}
			var status ApprovalResult
			if err := state.ReadCustomStatus(&status); err != nil {
				return false, err
			}
			return status.Status == "Pending", nil
		}); err != nil {
			return err
		}
		response := ApprovalResponse{
			IsApproved: decision, Approver: "Console Approver", Comments: "Automated demo response",
		}
		if err := c.RaiseEvent(ctx, id, approvalEvent, api.WithEventPayload(response)); err != nil {
			return err
		}
		want.Status = "Rejected"
		if *decision {
			want.Status = "Approved"
		}
		want.Approver = response.Approver
	}

	var result ApprovalResult
	if err := sample.Wait(ctx, c, id, &result); err != nil {
		return err
	}
	if err := sample.Require(result == want, "%s result = %+v, want %+v", scenario, result, want); err != nil {
		return err
	}
	return sample.PrintJSON(struct {
		Scenario string         `json:"scenario"`
		Result   ApprovalResult `json:"result"`
	}{scenario, result})
}

func run(ctx context.Context) error {
	r, err := newRegistry()
	if err != nil {
		return err
	}
	return sample.WithHost(ctx, r, func(ctx context.Context, c *dts.Client) error {
		approve, reject := true, false
		for _, scenario := range []struct {
			name     string
			decision *bool
		}{{"approve", &approve}, {"reject", &reject}, {"timeout", nil}} {
			if err := verifyRequest(ctx, c, scenario.name, scenario.decision); err != nil {
				return err
			}
		}
		return nil
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
	sample.Main("human-interaction", run)
}
