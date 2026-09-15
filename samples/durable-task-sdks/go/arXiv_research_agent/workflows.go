package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/microsoft/durabletask-go/task"
)

const (
	researchName   = "GoArxivResearch"
	paperName      = "GoArxivPaperResearch"
	searchName     = "GoArxivSearch"
	fetchName      = "GoArxivFetchPaper"
	analyzeName    = "GoArxivAnalyzePapers"
	decideName     = "GoArxivDecideContinuation"
	gapsName       = "GoArxivIdentifyGaps"
	synthesizeName = "GoArxivSynthesize"
)

func activityOptions(input any) []task.CallActivityOption {
	return []task.CallActivityOption{
		task.WithActivityInput(input),
		task.WithActivityRetryPolicy(&task.RetryPolicy{
			MaxAttempts: 3, InitialRetryInterval: time.Second,
			BackoffCoefficient: 2, MaxRetryInterval: 5 * time.Second, RetryTimeout: 2 * time.Minute,
		}),
	}
}

func researchOrchestrator(ctx *task.OrchestrationContext) (any, error) {
	var state researchState
	if err := ctx.GetInput(&state); err != nil {
		return nil, err
	}
	if err := validateState(state); err != nil {
		return nil, err
	}
	if state.Iteration >= state.MaxIterations {
		return finishResearch(ctx, state)
	}
	state.Iteration++
	if err := setProgress(ctx, state, "researching"); err != nil {
		return nil, err
	}
	// Schedule the entire query batch before awaiting: later iterations fan out.
	queries := make([]task.Task, len(state.Queries))
	for slot, query := range state.Queries {
		queries[slot] = ctx.CallSubOrchestrator(paperName,
			task.WithSubOrchestratorInput(queryInput{state.Topic, query, state.Mode, state.Iteration, slot}),
			task.WithSubOrchestrationInstanceID(fmt.Sprintf("%s-iteration-%d-query-%d", ctx.ID, state.Iteration, slot)))
	}
	results, err := collectFanoutResults[queryResult](ctx, queries)
	if err != nil {
		return nil, fmt.Errorf("drain research query fan-out: %w", err)
	}
	for _, result := range results {
		state.Findings = append(state.Findings, result.Finding)
		state.Papers = mergePapers(state.Papers, result.Papers)
	}
	if err := validateState(state); err != nil {
		return nil, err
	}
	if err := setProgress(ctx, state, "deciding"); err != nil {
		return nil, err
	}
	var shouldContinue bool
	if err := ctx.CallActivity(decideName, activityOptions(state)...).Await(&shouldContinue); err != nil {
		return nil, err
	}
	if !shouldContinue || state.Iteration >= state.MaxIterations || len(state.Papers) >= maxPapers {
		return finishResearch(ctx, state)
	}
	var nextQueries []string
	if err := ctx.CallActivity(gapsName, activityOptions(state)...).Await(&nextQueries); err != nil {
		return nil, err
	}
	nextQueries = freshQueries(nextQueries, state.Findings)
	if len(nextQueries) == 0 {
		return finishResearch(ctx, state)
	}
	state.Queries = nextQueries
	if err := validateState(state); err != nil {
		return nil, err
	}
	// Input contains all deterministic progress; no wall clock, network, or env
	// reads occur in either orchestrator. Each generation has a bounded history.
	ctx.ContinueAsNew(state)
	return nil, nil
}

func paperOrchestrator(ctx *task.OrchestrationContext) (any, error) {
	var input queryInput
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	if err := validateQuery(input.Query); err != nil {
		return nil, err
	}
	var ids []string
	if err := ctx.CallActivity(searchName, activityOptions(input)...).Await(&ids); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return queryResult{
			Finding: finding{Query: input.Query, PaperIDs: []string{}, analysis: analysis{
				Insights: []string{}, KeyPoints: []string{}, ResearchGaps: []string{}, Summary: "No papers found for this query.",
			}}, Papers: []paper{},
		}, nil
	}
	if len(ids) > 3 {
		return nil, errors.New("search exceeded its three-paper budget")
	}
	fetches := make([]task.Task, len(ids))
	for index, id := range ids {
		fetches[index] = ctx.CallActivity(fetchName, activityOptions(fetchInput{input.Mode, id})...)
	}
	papers, err := collectFanoutResults[paper](ctx, fetches)
	if err != nil {
		return nil, fmt.Errorf("drain paper fetch fan-out: %w", err)
	}
	var analysis finding
	if err := ctx.CallActivity(analyzeName, activityOptions(analysisInput{input.Mode, input.Topic, input.Query, papers})...).Await(&analysis); err != nil {
		return nil, err
	}
	return queryResult{Finding: analysis, Papers: papers}, nil
}

func collectFanoutResults[T any](ctx *task.OrchestrationContext, tasks []task.Task) ([]T, error) {
	// WhenAll drains failures too, so a failed parent cannot strand a sibling's
	// external/billable work. Decode only after the entire batch has completed.
	if err := ctx.WhenAll(tasks...); err != nil {
		return nil, err
	}
	results := make([]T, len(tasks))
	for index, pending := range tasks {
		if err := pending.Await(&results[index]); err != nil {
			return nil, err
		}
	}
	return results, nil
}

func finishResearch(ctx *task.OrchestrationContext, state researchState) (any, error) {
	if err := setProgress(ctx, state, "synthesizing"); err != nil {
		return nil, err
	}
	var report string
	if err := ctx.CallActivity(synthesizeName, activityOptions(state)...).Await(&report); err != nil {
		return nil, err
	}
	if err := setProgress(ctx, state, "completed"); err != nil {
		return nil, err
	}
	return researchResult{
		Topic: state.Topic, Mode: state.Mode, Iterations: state.Iteration,
		FindingsCount: len(state.Findings), PaperIDs: paperIDs(state.Papers),
		Report: report, Findings: state.Findings, Papers: state.Papers,
	}, nil
}

func setProgress(ctx *task.OrchestrationContext, state researchState, phase string) error {
	return ctx.SetCustomStatusValue(progress{state.Mode, phase, state.Iteration, len(state.Findings), paperIDs(state.Papers)})
}

func mergePapers(existing, added []paper) []paper {
	byID := make(map[string]paper, len(existing)+len(added))
	for _, item := range append(append([]paper{}, existing...), added...) {
		byID[item.ID] = item
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]paper, 0, len(ids))
	for _, id := range ids {
		result = append(result, byID[id])
	}
	return result
}

func paperIDs(papers []paper) []string {
	ids := make([]string, 0, len(papers))
	for _, paper := range papers {
		ids = append(ids, paper.ID)
	}
	return ids
}

func freshQueries(queries []string, findings []finding) []string {
	seen := map[string]bool{}
	for _, finding := range findings {
		seen[strings.ToLower(finding.Query)] = true
	}
	result := []string{}
	for _, query := range queries {
		query = strings.TrimSpace(query)
		key := strings.ToLower(query)
		if query != "" && !seen[key] {
			result = append(result, query)
			seen[key] = true
		}
		if len(result) == 2 {
			break
		}
	}
	return result
}
