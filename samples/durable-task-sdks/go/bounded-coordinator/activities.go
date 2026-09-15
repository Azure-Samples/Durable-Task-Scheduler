package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/microsoft/durabletask-go/task"
)

const (
	totalBatches  = 3
	itemsPerBatch = 5
	sourceLimit   = 50
)

type BatchRequest struct {
	Cursor   string `json:"cursor"`
	MaxItems int    `json:"max_items"`
}

type Item struct {
	ID       string `json:"id"`
	TenantID string `json:"tenant_id"`
	Payload  string `json:"payload"`
}

type Batch struct {
	Items      []Item `json:"items"`
	NextCursor string `json:"next_cursor"`
	HasMore    bool   `json:"has_more"`
}

func cursorFor(batch int) string {
	if batch == 0 {
		return ""
	}
	return fmt.Sprintf("cursor-%d", batch)
}

func nextBatch(input BatchRequest) (Batch, error) {
	if input.MaxItems < itemsPerBatch || input.MaxItems > sourceLimit {
		return Batch{}, fmt.Errorf("max_items must be between %d and %d; fixture cursors advance by whole pages",
			itemsPerBatch, sourceLimit)
	}
	previous := 0
	if input.Cursor != "" {
		raw, ok := strings.CutPrefix(input.Cursor, "cursor-")
		if !ok {
			return Batch{}, fmt.Errorf("invalid cursor %q", input.Cursor)
		}
		var err error
		previous, err = strconv.Atoi(raw)
		if err != nil || previous < 1 || previous > totalBatches || input.Cursor != cursorFor(previous) {
			return Batch{}, fmt.Errorf("invalid cursor %q", input.Cursor)
		}
	}
	if previous == totalBatches {
		return Batch{Items: []Item{}}, nil
	}
	batchNumber := previous + 1
	batch := Batch{
		Items:      make([]Item, itemsPerBatch),
		NextCursor: cursorFor(batchNumber), HasMore: batchNumber < totalBatches,
	}
	for i := range batch.Items {
		batch.Items[i] = Item{
			ID: fmt.Sprintf("item-%d-%d", batchNumber, i+1), TenantID: fmt.Sprintf("tenant-%d", i+1),
			Payload: fmt.Sprintf("data-%d-%d", batchNumber, i+1),
		}
	}
	return batch, nil
}

func getNextBatch(ctx task.ActivityContext) (any, error) {
	var input BatchRequest
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	// Simulation only: the cursor addresses a stateless, finite source fixture.
	return nextBatch(input)
}

func applyChange(ctx task.ActivityContext) (any, error) {
	var item Item
	if err := ctx.GetInput(&item); err != nil {
		return nil, err
	}
	if item.ID == "" || item.TenantID == "" || item.Payload == "" {
		return nil, errors.New("change requires an item ID, tenant ID, and payload")
	}
	// Simulation only: no tenant data is changed.
	return "processed:" + item.ID, nil
}
