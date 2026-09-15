package main

import (
	"fmt"

	"github.com/microsoft/durabletask-go/task"
)

const (
	maxItems     = 100
	maxMagnitude = 1_000_000
)

type WorkResult struct {
	Item   int64 `json:"item"`
	Result int64 `json:"result"`
}

type Summary struct {
	TotalItems int     `json:"total_items"`
	Sum        int64   `json:"sum"`
	Average    float64 `json:"average"`
}

func square(item int64) (WorkResult, error) {
	if item < -maxMagnitude || item > maxMagnitude {
		return WorkResult{}, fmt.Errorf("item %d exceeds the sample's safe arithmetic range", item)
	}
	return WorkResult{Item: item, Result: item * item}, nil
}

func processWorkItem(ctx task.ActivityContext) (any, error) {
	var item int64
	if err := ctx.GetInput(&item); err != nil {
		return nil, err
	}
	return square(item)
}

func summarize(results []WorkResult) (Summary, error) {
	if len(results) > maxItems {
		return Summary{}, fmt.Errorf("batch contains more than %d items", maxItems)
	}
	summary := Summary{TotalItems: len(results)}
	for _, result := range results {
		expected, err := square(result.Item)
		if err != nil {
			return Summary{}, err
		}
		if result != expected {
			return Summary{}, fmt.Errorf("incorrect square for item %d: %d", result.Item, result.Result)
		}
		summary.Sum += result.Result
	}
	if len(results) != 0 {
		summary.Average = float64(summary.Sum) / float64(len(results))
	}
	return summary, nil
}

func aggregateResults(ctx task.ActivityContext) (any, error) {
	var results []WorkResult
	if err := ctx.GetInput(&results); err != nil {
		return nil, err
	}
	return summarize(results)
}
