package main

import (
	"testing"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/testutil"
)

func TestIntegration(t *testing.T) {
	ctx := testutil.IntegrationContext(t)
	a, b, err := newRegistries()
	if err != nil {
		t.Fatal(err)
	}
	workerA, err := sample.Start(ctx, a, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := workerA.Close(); err != nil {
			t.Error(err)
		}
	})
	workerB, err := sample.Start(ctx, b, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := workerB.Close(); err != nil {
			t.Error(err)
		}
	})

	greeting, sum, err := runWorkloads(ctx, workerA.Client)
	if err != nil {
		t.Fatal(err)
	}
	if err := testutil.Require(greeting == (greetingResult{Worker: "A", Result: "Hello, World!"}),
		"greeting routed incorrectly: %+v", greeting); err != nil {
		t.Error(err)
	}
	if err := testutil.Require(sum == (mathResult{Worker: "B", Result: 42}),
		"math routed incorrectly: %+v", sum); err != nil {
		t.Error(err)
	}
}
