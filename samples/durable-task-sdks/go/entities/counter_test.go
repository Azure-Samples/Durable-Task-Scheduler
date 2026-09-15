package main

import (
	"testing"
	"time"

	"github.com/microsoft/durabletask-go/task"
)

func TestCounterOperations(t *testing.T) {
	var state counterState
	for _, operation := range []struct {
		name   string
		amount int
		want   int
	}{{"add", 100, 100}, {"subtract", 25, 75}, {"subtract", 100, -25}, {"add", 37, 12}} {
		if err := state.change(operation.name, operation.amount, time.Time{}); err != nil {
			t.Fatal(err)
		}
		if state.Value != operation.want {
			t.Fatalf("%s: got %d, want %d", operation.name, state.Value, operation.want)
		}
	}
	now := time.Date(2026, 1, 1, 0, 0, 5, 0, time.UTC)
	if err := state.change("reset", 0, now); err != nil {
		t.Fatal(err)
	}
	if state.Value != 0 || !state.ResetAt.Equal(now) {
		t.Fatalf("reset state = %+v", state)
	}
	if err := state.change("unknown", 123, now); err == nil {
		t.Fatal("unknown operation succeeded")
	}
}

func TestCounterStateCallsAndDeletion(t *testing.T) {
	ctx := &task.EntityContext{Operation: "get"}
	got, err := counter(ctx)
	if err != nil || got != 0 || ctx.HasState() {
		t.Fatalf("initial get = %v, %v; has state = %v", got, err, ctx.HasState())
	}
	if err := ctx.SetState(counterState{Value: 12}); err != nil {
		t.Fatal(err)
	}
	got, err = counter(ctx)
	if err != nil || got != 12 {
		t.Fatalf("get = %v, %v", got, err)
	}
	ctx.Operation = "snapshot"
	got, err = counter(ctx)
	if err != nil || got.(counterState).Value != 12 {
		t.Fatalf("snapshot = %v, %v", got, err)
	}
	ctx.Operation = "delete"
	if _, err := counter(ctx); err != nil || ctx.HasState() {
		t.Fatalf("delete: %v, has state = %v", err, ctx.HasState())
	}
	ctx.Operation = "add"
	if _, err := counter(ctx); err == nil {
		t.Fatal("add without an input succeeded")
	}
}

func TestScheduledSignalVerification(t *testing.T) {
	due := time.Date(2026, 1, 1, 0, 0, 5, 0, time.UTC)
	valid := workflowResult{Before: 12, After: 0, ReadAt: due.Add(-time.Second), DueAt: due, ResetAt: due}
	if err := validateResult(valid); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		change func(*workflowResult)
	}{
		{"delivered early", func(r *workflowResult) { r.ResetAt = due.Add(-time.Nanosecond) }},
		{"never delivered", func(r *workflowResult) { r.ResetAt = time.Time{} }},
		{"late before read", func(r *workflowResult) { r.ReadAt = due }},
		{"incorrect arithmetic", func(r *workflowResult) { r.Before = 13 }},
		{"not reset", func(r *workflowResult) { r.After = 12 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := valid
			test.change(&result)
			if err := validateResult(result); err == nil {
				t.Fatal("invalid result passed verification")
			}
		})
	}
}

func TestEntityRegistry(t *testing.T) {
	registry, err := newRegistry()
	if err != nil {
		t.Fatal(err)
	}
	snapshot := registry.Snapshot()
	if len(snapshot.Entities) != 1 || snapshot.Entities[0] != counterName ||
		len(snapshot.Orchestrators) != 1 || snapshot.Orchestrators[0].Name != workflowName ||
		len(snapshot.Activities) != 0 {
		t.Fatalf("unexpected registry: %+v", snapshot)
	}
}
