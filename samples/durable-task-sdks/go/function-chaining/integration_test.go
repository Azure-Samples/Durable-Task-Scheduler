package main

import (
	"context"
	"testing"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/testutil"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
)

func TestIntegration(t *testing.T) {
	ctx := testutil.IntegrationContext(t)
	r, err := newRegistry()
	if err != nil {
		t.Fatal(err)
	}
	err = sample.WithHost(ctx, r, func(ctx context.Context, c *dts.Client) (err error) {
		id, err := c.ScheduleNewOrchestration(ctx, orchestrationName,
			api.WithInstanceID(sample.ID("function-chaining-test")), api.WithInput("User"))
		if err != nil {
			return err
		}
		defer stopOnError(c, id, &err)

		var output string
		if err := sample.Wait(ctx, c, id, &output); err != nil {
			return err
		}
		const want = "Hello User! How are you today? I hope you're doing well!"
		return testutil.Require(output == want, "greeting = %q, want %q", output, want)
	})
	if err != nil {
		t.Fatal(err)
	}
}
