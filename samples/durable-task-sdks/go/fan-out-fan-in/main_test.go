package main

import (
	"context"
	"encoding/json"
	"math"
	"testing"
)

type activityInput []byte

func (input activityInput) GetInput(target any) error { return json.Unmarshal(input, target) }
func (activityInput) Context() context.Context        { return context.Background() }

func TestSquaresAndAggregation(t *testing.T) {
	results := make([]WorkResult, 10)
	for i := range results {
		raw, err := json.Marshal(i + 1)
		if err != nil {
			t.Fatal(err)
		}
		output, err := processWorkItem(activityInput(raw))
		if err != nil {
			t.Fatal(err)
		}
		results[i] = output.(WorkResult)
	}
	raw, err := json.Marshal(results)
	if err != nil {
		t.Fatal(err)
	}
	output, err := aggregateResults(activityInput(raw))
	if err != nil {
		t.Fatal(err)
	}
	if want := (Summary{TotalItems: 10, Sum: 385, Average: 38.5}); output != want {
		t.Fatalf("summary = %+v, want %+v", output, want)
	}
}

func TestSquareBounds(t *testing.T) {
	for _, item := range []int64{0, -3, maxMagnitude, -maxMagnitude} {
		got, err := square(item)
		if err != nil || got != (WorkResult{Item: item, Result: item * item}) {
			t.Fatalf("square(%d) = %+v, %v", item, got, err)
		}
	}
	for _, item := range []int64{maxMagnitude + 1, -maxMagnitude - 1, math.MaxInt64, math.MinInt64} {
		if _, err := square(item); err == nil {
			t.Fatalf("unsafe item %d was accepted", item)
		}
	}
}

func TestAggregationEdgeCases(t *testing.T) {
	if got, err := summarize(nil); err != nil || got != (Summary{}) {
		t.Fatalf("empty summary = %+v, %v", got, err)
	}
	if got, err := summarize([]WorkResult{{Item: -2, Result: 4}, {Item: -2, Result: 4}}); err != nil ||
		got != (Summary{TotalItems: 2, Sum: 8, Average: 4}) {
		t.Fatalf("duplicate/negative summary = %+v, %v", got, err)
	}
	if _, err := summarize([]WorkResult{{Item: 3, Result: 8}}); err == nil {
		t.Fatal("incorrect result accepted")
	}
	if _, err := summarize(make([]WorkResult, maxItems+1)); err == nil {
		t.Fatal("unbounded batch accepted")
	}
	if _, err := processWorkItem(activityInput(`"not a number"`)); err == nil {
		t.Fatal("malformed work item accepted")
	}
	if _, err := aggregateResults(activityInput(`{}`)); err == nil {
		t.Fatal("malformed result list accepted")
	}
}
