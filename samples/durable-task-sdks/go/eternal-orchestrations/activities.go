package main

import (
	"errors"
	"fmt"

	"github.com/microsoft/durabletask-go/task"
)

type CleanupReceipt struct {
	Iteration int      `json:"iteration"`
	Removed   []string `json:"removed"`
	Retained  []string `json:"retained"`
	Message   string   `json:"message"`
}

func cleanupFixture(iteration int) (CleanupReceipt, error) {
	if iteration < 1 || iteration > iterations {
		return CleanupReceipt{}, errors.New("cleanup iteration is outside the fixture")
	}
	// Simulation only: partition in-memory records; never delete user files or data.
	records := []struct {
		id      string
		expired bool
	}{
		{fmt.Sprintf("expired-%d-a", iteration), true},
		{fmt.Sprintf("current-%d", iteration), false},
		{fmt.Sprintf("expired-%d-b", iteration), true},
	}
	receipt := CleanupReceipt{Iteration: iteration, Message: "Cleanup completed"}
	for _, record := range records {
		if record.expired {
			receipt.Removed = append(receipt.Removed, record.id)
		} else {
			receipt.Retained = append(receipt.Retained, record.id)
		}
	}
	return receipt, nil
}

func cleanupTask(ctx task.ActivityContext) (any, error) {
	var iteration int
	if err := ctx.GetInput(&iteration); err != nil {
		return nil, err
	}
	return cleanupFixture(iteration)
}
