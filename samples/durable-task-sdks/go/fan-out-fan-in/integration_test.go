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
	err = sample.WithHost(ctx, r, func(ctx context.Context, c *dts.Client) error {
		if err := verifyBatch(ctx, c, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			Summary{TotalItems: 10, Sum: 385, Average: 38.5}); err != nil {
			return err
		}
		return verifyBatch(ctx, c, []int64{}, Summary{})
	})
	if err != nil {
		t.Fatal(err)
	}
}

func verifyBatch(ctx context.Context, c *dts.Client, items []int64, want Summary) (err error) {
	id, err := c.ScheduleNewOrchestration(ctx, orchestrationName,
		api.WithInstanceID(sample.ID("fan-out-fan-in-test")), api.WithInput(items))
	if err != nil {
		return err
	}
	defer stopOnError(c, id, &err)

	var got Summary
	if err := sample.Wait(ctx, c, id, &got); err != nil {
		return err
	}
	return testutil.Require(got == want, "aggregation = %+v, want %+v", got, want)
}
