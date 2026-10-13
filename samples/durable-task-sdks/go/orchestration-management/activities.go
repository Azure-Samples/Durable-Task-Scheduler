package main

import (
	"errors"

	"github.com/microsoft/durabletask-go/task"
)

func processBatch(ctx task.ActivityContext) (any, error) {
	var input batchInput
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	if input.BatchID == "" || input.ItemCount < 0 {
		return nil, errors.New("batch_id must be nonempty and item_count must be nonnegative")
	}
	return batchResult{BatchID: input.BatchID, ItemsProcessed: input.ItemCount, Status: "success"}, nil
}
