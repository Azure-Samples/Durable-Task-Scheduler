package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/microsoft/durabletask-go/task"
)

type activityInput []byte

func (input activityInput) GetInput(target any) error { return json.Unmarshal(input, target) }
func (activityInput) Context() context.Context        { return context.Background() }

func TestApprovalOutcomes(t *testing.T) {
	approve, reject := true, false
	for _, scenario := range []struct {
		name     string
		response *ApprovalResponse
		want     ApprovalResult
	}{
		{"approved", &ApprovalResponse{IsApproved: &approve, Approver: "Alex"},
			ApprovalResult{RequestID: "request-1", Status: "Approved", Approver: "Alex"}},
		{"rejected", &ApprovalResponse{IsApproved: &reject, Approver: "Alex"},
			ApprovalResult{RequestID: "request-1", Status: "Rejected", Approver: "Alex"}},
		{"timeout", nil, ApprovalResult{RequestID: "request-1", Status: "Timeout"}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			got, err := approvalOutcome("request-1", scenario.response)
			if err != nil || got != scenario.want {
				t.Fatalf("outcome = %+v, %v; want %+v", got, err, scenario.want)
			}
			if scenario.response != nil {
				data, err := json.Marshal(ProcessInput{RequestID: "request-1", Response: *scenario.response})
				if err != nil {
					t.Fatal(err)
				}
				output, err := processApproval(activityInput(data))
				if err != nil || output != scenario.want {
					t.Fatalf("activity result = %+v, %v; want %+v", output, err, scenario.want)
				}
			}
		})
	}
}

func TestApprovalValidation(t *testing.T) {
	approved := true
	for _, response := range []*ApprovalResponse{
		{},
		{IsApproved: &approved},
		{Approver: "Alex"},
	} {
		if _, err := approvalOutcome("request-1", response); err == nil {
			t.Fatalf("incomplete response accepted: %+v", response)
		}
	}
	if _, err := approvalOutcome("", nil); err == nil {
		t.Fatal("empty request ID accepted")
	}
	valid := ApprovalRequest{RequestID: "request-1", Requester: "User", Item: "Vacation Request", TimeoutSeconds: 1}
	if err := valid.validate(); err != nil {
		t.Fatal(err)
	}
	for _, timeout := range []int{-1, 0, 24*60*60 + 1} {
		invalid := valid
		invalid.TimeoutSeconds = timeout
		if err := invalid.validate(); err == nil {
			t.Fatalf("invalid timeout %d accepted", timeout)
		}
	}
	for _, activity := range []task.Activity{submitApprovalRequest, processApproval} {
		if _, err := activity(activityInput(`{`)); err == nil {
			t.Fatal("malformed input accepted")
		}
	}
}

func TestSubmitApproval(t *testing.T) {
	output, err := submitApprovalRequest(activityInput(
		`{"request_id":"request-1","requester":"User","item":"Vacation Request","timeout_seconds":10}`))
	want := ApprovalResult{RequestID: "request-1", Status: "Pending"}
	if err != nil || output != want {
		t.Fatalf("submission = %+v, %v; want %+v", output, err, want)
	}
	if _, err := submitApprovalRequest(activityInput(`{"timeout_seconds":1}`)); err == nil {
		t.Fatal("incomplete request accepted")
	}
}
