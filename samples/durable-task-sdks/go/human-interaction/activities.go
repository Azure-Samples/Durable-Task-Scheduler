package main

import (
	"errors"
	"strings"

	"github.com/microsoft/durabletask-go/task"
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
