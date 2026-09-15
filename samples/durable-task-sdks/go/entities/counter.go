package main

import (
	"fmt"
	"time"

	"github.com/microsoft/durabletask-go/task"
)

type counterState struct {
	Value   int       `json:"value"`
	ResetAt time.Time `json:"reset_at,omitempty"`
}

func counter(ctx *task.EntityContext) (any, error) {
	var state counterState
	if ctx.HasState() {
		if err := ctx.GetState(&state); err != nil {
			return nil, err
		}
	}
	switch ctx.Operation {
	case "get":
		return state.Value, nil
	case "snapshot":
		return state, nil
	case "delete":
		ctx.DeleteState()
		return nil, nil
	}
	var amount int
	if ctx.Operation == "add" || ctx.Operation == "subtract" {
		if err := ctx.GetInput(&amount); err != nil {
			return nil, err
		}
	}
	if err := state.change(ctx.Operation, amount, ctx.CurrentTimeUTC()); err != nil {
		return nil, err
	}
	if err := ctx.SetState(state); err != nil {
		return nil, err
	}
	return state.Value, nil
}

func (s *counterState) change(operation string, amount int, now time.Time) error {
	switch operation {
	case "add":
		s.Value += amount
	case "subtract":
		s.Value -= amount
	case "reset":
		s.Value = 0
		// This records execution time, not the requested delivery time.
		s.ResetAt = now
	default:
		return fmt.Errorf("unknown counter operation %q", operation)
	}
	return nil
}
