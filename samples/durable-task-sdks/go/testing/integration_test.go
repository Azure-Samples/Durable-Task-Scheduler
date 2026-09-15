package main

import (
	"strings"
	"testing"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/testutil"
	"github.com/microsoft/durabletask-go/api"
)

func TestIntegration(t *testing.T) {
	ctx := testutil.IntegrationContext(t)
	r, err := registry()
	if err != nil {
		t.Fatal(err)
	}
	host, err := sample.Start(ctx, r, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := host.Close(); err != nil {
			t.Error(err)
		}
	})

	cases := []struct {
		name  string
		input order
		want  orderResult
		cause string
	}{
		{"single", order{"Alice", []item{{"Widget", 2, 1000}}}, orderResult{"PAY-2000", "TRACK-ALICE-1", 2000, "completed"}, ""},
		{"multiple", order{"Bob", []item{{"Widget", 3, 2500}, {"Gadget", 1, 9999}}}, orderResult{"PAY-17499", "TRACK-BOB-2", 17499, "completed"}, ""},
		{"missing-customer", order{"", []item{{"Widget", 1, 1000}}}, orderResult{}, "customer name"},
		{"empty", order{"Eve", nil}, orderResult{}, "at least one item"},
		{"invalid-quantity", order{"Mallory", []item{{"Widget", 0, 1000}}}, orderResult{}, "invalid quantity"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			id, err := host.Client.ScheduleNewOrchestration(ctx, orderWorkflowName,
				api.WithInstanceID(sample.ID("testing-"+test.name)), api.WithInput(test.input))
			if err != nil {
				t.Fatal(err)
			}
			if test.cause == "" {
				var result orderResult
				if err := sample.Wait(ctx, host.Client, id, &result); err != nil {
					t.Fatal(err)
				}
				if result != test.want {
					t.Fatalf("got %+v, want %+v", result, test.want)
				}
				return
			}
			metadata, err := host.Client.WaitForOrchestrationCompletion(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			if metadata.RuntimeStatus != api.RUNTIME_STATUS_FAILED || !hasCause(metadata.FailureDetails, test.cause) {
				t.Fatalf("expected failure containing %q, got %s: %v", test.cause, metadata.RuntimeStatus, metadata.FailureDetails)
			}
		})
	}
}

func hasCause(details *api.FailureDetails, text string) bool {
	for current := details; current != nil; current = current.InnerFailure {
		if strings.Contains(current.ErrorMessage, text) {
			return true
		}
	}
	return false
}
