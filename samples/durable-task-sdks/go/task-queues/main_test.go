package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/microsoft/durabletask-go/task"
)

type localActivityContext struct {
	input string
}

func (c localActivityContext) GetInput(target any) error {
	return json.Unmarshal([]byte(c.input), target)
}

func (c localActivityContext) Context() context.Context {
	return context.Background()
}

func TestActivities(t *testing.T) {
	for _, activity := range []struct {
		name string
		call task.Activity
		want string
	}{
		{"accept", acceptOrder, "accepted:order-1001"},
		{"partner", callPartner, "partner-confirmed:order-1001"},
	} {
		t.Run(activity.name, func(t *testing.T) {
			for _, input := range []string{`{"id":"order-1001"}`, `{"id":" "}`, `{`} {
				output, err := activity.call(localActivityContext{input})
				if input == `{"id":"order-1001"}` {
					if err != nil || output != activity.want {
						t.Fatalf("output = %v, error = %v", output, err)
					}
				} else if err == nil {
					t.Fatalf("expected error for input %q", input)
				}
			}
		})
	}
}

func TestResultValidation(t *testing.T) {
	input := order{ID: "order-1001"}
	output := orderResult{input.ID, "accepted:" + input.ID,
		"partner-confirmed:" + input.ID, "partner-confirmed:" + input.ID}
	if err := checkResult(input, output); err != nil {
		t.Fatal(err)
	}
	output.ChildPartner = "partner-confirmed:wrong-order"
	if err := checkResult(input, output); err == nil || !strings.Contains(err.Error(), "wrong-order") {
		t.Fatalf("expected mismatched child result error, got %v", err)
	}
}
