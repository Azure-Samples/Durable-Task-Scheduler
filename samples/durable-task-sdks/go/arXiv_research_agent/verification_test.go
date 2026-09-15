package main

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/testutil"
	"github.com/microsoft/durabletask-go/api"
)

type historyReader interface {
	GetOrchestrationHistory(context.Context, api.InstanceID, api.HistoryQuery) (*api.OrchestrationHistory, error)
}

func verifyFixtureCheckpoint(ctx context.Context, client historyReader, metadata *api.OrchestrationMetadata) error {
	if metadata == nil || metadata.InstanceID == "" || metadata.ExecutionID == "" || metadata.Name != researchName {
		return errors.New("checkpoint verification requires the research instance and its current execution ID")
	}
	// DTS metadata can retain the original input after ContinueAsNew. Only the
	// selected execution's ExecutionStarted input proves the persisted checkpoint.
	history, err := client.GetOrchestrationHistory(ctx, metadata.InstanceID, api.HistoryQuery{
		ExecutionID: metadata.ExecutionID, MaxEvents: 200, MaxBytes: 1024 * 1024,
	})
	if err != nil {
		return fmt.Errorf("read current research execution history: %w", err)
	}
	if history == nil || history.InstanceID != metadata.InstanceID || history.ExecutionID != metadata.ExecutionID {
		return errors.New("checkpoint history does not match the selected research execution")
	}
	var checkpoint researchState
	starts := 0
	for _, event := range history.Events {
		if event == nil {
			return errors.New("nil event in research checkpoint history")
		}
		if event.Type != api.HistoryEventExecutionStarted {
			continue
		}
		starts++
		if starts > 1 {
			return errors.New("research checkpoint history contains multiple execution starts")
		}
		start := event.ExecutionStarted
		if start == nil || start.Name != researchName || start.InstanceID != metadata.InstanceID ||
			start.ExecutionID != metadata.ExecutionID || start.SerializedInput == "" {
			return errors.New("research execution start has missing or mismatched identity/input")
		}
		if err := event.ReadInput(&checkpoint); err != nil {
			return fmt.Errorf("decode research execution checkpoint: %w", err)
		}
	}
	if starts != 1 {
		return errors.New("research checkpoint history has no execution start")
	}
	if err := validateState(checkpoint); err != nil {
		return fmt.Errorf("invalid research execution checkpoint: %w", err)
	}
	expectedFinding := finding{
		Query: demoTopic, PaperIDs: []string{"fixture-001", "fixture-002"},
		analysis: analysis{
			Insights: []string{"This is fixture evidence, not an academic claim."}, RelevanceScore: 8,
			Summary:      "Synthetic analysis of fixture-001, fixture-002.",
			KeyPoints:    []string{"Exercise checkpointing, idempotency and recovery."},
			ResearchGaps: []string{"Real-world evidence remains unverified."},
		},
	}
	if checkpoint.Topic != demoTopic || checkpoint.Mode != "fixture" || checkpoint.MaxIterations != 2 ||
		checkpoint.Iteration != 1 || !reflect.DeepEqual(checkpoint.Findings, []finding{expectedFinding}) ||
		!reflect.DeepEqual(checkpoint.Queries, []string{demoTopic + " methods", demoTopic + " evaluation"}) ||
		!reflect.DeepEqual(paperIDs(checkpoint.Papers), []string{"fixture-001", "fixture-002"}) {
		return fmt.Errorf("continue-as-new checkpoint was not preserved: %+v", checkpoint)
	}
	for _, item := range checkpoint.Papers {
		expected, err := fixturePaper(item.ID)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(item, expected) {
			return fmt.Errorf("continued checkpoint lost fetched fixture metadata: %+v", item)
		}
	}
	return nil
}

func verifyFixtureResult(result researchResult) error {
	if err := testutil.Require(result.Mode == "fixture" && result.Topic == demoTopic && result.Iterations == 2 &&
		result.FindingsCount == 3 && len(result.Findings) == 3 && len(result.Papers) == 3 &&
		reflect.DeepEqual(result.PaperIDs, []string{"fixture-001", "fixture-002", "fixture-003"}) &&
		result.Report == expectedDemoReport, "fixture report, iterations or paper IDs differ: %+v", result); err != nil {
		return err
	}
	for _, item := range result.Papers {
		expected, err := fixturePaper(item.ID)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(item, expected) {
			return fmt.Errorf("fetched fixture metadata differs: %+v", item)
		}
	}
	expectedQueries := []string{demoTopic, demoTopic + " methods", demoTopic + " evaluation"}
	expectedIDs := [][]string{{"fixture-001", "fixture-002"}, {"fixture-002", "fixture-003"}, {"fixture-001", "fixture-003"}}
	for index, finding := range result.Findings {
		if finding.Query != expectedQueries[index] || finding.RelevanceScore != 8 ||
			!reflect.DeepEqual(finding.PaperIDs, expectedIDs[index]) ||
			finding.Summary != "Synthetic analysis of "+strings.Join(expectedIDs[index], ", ")+"." {
			return fmt.Errorf("unexpected fixture analysis: %+v", finding)
		}
	}
	return nil
}

const expectedDemoReport = `# Fixture research report

> Synthetic fixture only: no arXiv search or model inference was performed.

## Summary
Topic: durable workflow reliability
Completed 2 iterations with 3 query analyses and 3 synthetic papers.

## Key Findings
- durable workflow reliability: Synthetic analysis of fixture-001, fixture-002.
- durable workflow reliability methods: Synthetic analysis of fixture-002, fixture-003.
- durable workflow reliability evaluation: Synthetic analysis of fixture-001, fixture-003.

## Methods & Approaches
Deterministic replay; idempotent activities; failure-injection tests (synthetic examples).

## Open Questions
Validate all synthetic claims against real papers before academic use.

## References
- fixture-001: Checkpointed workflows (synthetic; not an arXiv paper).
- fixture-002: Idempotent work execution (synthetic; not an arXiv paper).
- fixture-003: Recovery experiments (synthetic; not an arXiv paper).
`
