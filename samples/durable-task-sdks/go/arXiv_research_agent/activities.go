package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/microsoft/durabletask-go/task"
)

type paperSource interface {
	Search(context.Context, string) ([]string, error)
	Fetch(context.Context, string) (paper, error)
}

type researchModel interface {
	Analyze(context.Context, analysisInput) (analysis, error)
	Decide(context.Context, researchState) (bool, error)
	Gaps(context.Context, researchState) ([]string, error)
	Synthesize(context.Context, researchState) (string, error)
}

type activities struct {
	mode   string
	source paperSource
	model  researchModel
}

func newRegistry(a *activities) (*task.TaskRegistry, error) {
	registry := task.NewTaskRegistry()
	for _, item := range []struct {
		name string
		fn   task.Orchestrator
	}{{researchName, researchOrchestrator}, {paperName, paperOrchestrator}} {
		if err := registry.AddOrchestratorN(item.name, item.fn); err != nil {
			return nil, err
		}
	}
	for _, item := range []struct {
		name string
		fn   task.Activity
	}{
		{searchName, a.search}, {fetchName, a.fetch}, {analyzeName, a.analyze},
		{decideName, a.decide}, {gapsName, a.gaps}, {synthesizeName, a.synthesize},
	} {
		if err := registry.AddActivityN(item.name, item.fn); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func (a *activities) checkMode(mode string) error {
	if mode != a.mode {
		return errors.New("persisted research mode differs from worker mode; run the matching worker")
	}
	if mode == "real" && (a.source == nil || a.model == nil) {
		return errors.New("real research requires arXiv and Azure OpenAI clients")
	}
	if mode != "real" && mode != "fixture" {
		return errors.New("unknown research mode")
	}
	return nil
}

func (a *activities) search(ctx task.ActivityContext) (any, error) {
	var input queryInput
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	if err := a.checkMode(input.Mode); err != nil {
		return nil, err
	}
	if err := validateQuery(input.Query); err != nil {
		return nil, err
	}
	if input.Mode == "fixture" {
		if input.Iteration == 1 {
			return []string{"fixture-001", "fixture-002"}, nil
		}
		if input.Slot == 0 {
			return []string{"fixture-002", "fixture-003"}, nil
		}
		return []string{"fixture-001", "fixture-003"}, nil
	}
	callCtx, cancel := context.WithTimeout(ctx.Context(), 45*time.Second)
	defer cancel()
	return a.source.Search(callCtx, input.Query)
}

func (a *activities) fetch(ctx task.ActivityContext) (any, error) {
	var input fetchInput
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	if err := a.checkMode(input.Mode); err != nil {
		return nil, err
	}
	if input.Mode == "fixture" {
		return fixturePaper(input.ID)
	}
	callCtx, cancel := context.WithTimeout(ctx.Context(), 45*time.Second)
	defer cancel()
	return a.source.Fetch(callCtx, input.ID)
}

func (a *activities) analyze(ctx task.ActivityContext) (any, error) {
	var input analysisInput
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	if err := a.checkMode(input.Mode); err != nil {
		return nil, err
	}
	if len(input.Papers) == 0 || len(input.Papers) > 3 {
		return nil, errors.New("analysis requires one to three fetched papers")
	}
	var result analysis
	if input.Mode == "fixture" {
		result = analysis{
			Insights: []string{"This is fixture evidence, not an academic claim."}, RelevanceScore: 8,
			Summary:      "Synthetic analysis of " + strings.Join(paperIDs(input.Papers), ", ") + ".",
			KeyPoints:    []string{"Exercise checkpointing, idempotency and recovery."},
			ResearchGaps: []string{"Real-world evidence remains unverified."},
		}
	} else {
		callCtx, cancel := context.WithTimeout(ctx.Context(), 30*time.Second)
		defer cancel()
		var err error
		result, err = a.model.Analyze(callCtx, input)
		if err != nil {
			return nil, err
		}
	}
	if err := validateAnalysis(result); err != nil {
		return nil, err
	}
	return finding{Query: input.Query, analysis: result, PaperIDs: paperIDs(input.Papers)}, nil
}

func (a *activities) decide(ctx task.ActivityContext) (any, error) {
	var state researchState
	if err := ctx.GetInput(&state); err != nil {
		return nil, err
	}
	if err := a.checkMode(state.Mode); err != nil {
		return nil, err
	}
	if state.Iteration >= state.MaxIterations || len(state.Papers) >= maxPapers {
		return false, nil
	}
	if state.Mode == "fixture" {
		return true, nil
	}
	callCtx, cancel := context.WithTimeout(ctx.Context(), 30*time.Second)
	defer cancel()
	return a.model.Decide(callCtx, state)
}

func (a *activities) gaps(ctx task.ActivityContext) (any, error) {
	var state researchState
	if err := ctx.GetInput(&state); err != nil {
		return nil, err
	}
	if err := a.checkMode(state.Mode); err != nil {
		return nil, err
	}
	if state.Mode == "fixture" {
		if state.Iteration == 1 {
			return []string{state.Topic + " methods", state.Topic + " evaluation"}, nil
		}
		return []string{
			fmt.Sprintf("%s follow-up %d methods", state.Topic, state.Iteration),
			fmt.Sprintf("%s follow-up %d evaluation", state.Topic, state.Iteration),
		}, nil
	}
	callCtx, cancel := context.WithTimeout(ctx.Context(), 30*time.Second)
	defer cancel()
	queries, err := a.model.Gaps(callCtx, state)
	if err != nil {
		return nil, err
	}
	if len(queries) > 2 {
		return nil, errors.New("model returned more than two follow-up queries")
	}
	for _, query := range queries {
		if err := validateQuery(query); err != nil {
			return nil, err
		}
	}
	return queries, nil
}

func (a *activities) synthesize(ctx task.ActivityContext) (any, error) {
	var state researchState
	if err := ctx.GetInput(&state); err != nil {
		return nil, err
	}
	if err := a.checkMode(state.Mode); err != nil {
		return nil, err
	}
	if state.Mode == "fixture" {
		return fixtureReport(state), nil
	}
	callCtx, cancel := context.WithTimeout(ctx.Context(), 30*time.Second)
	defer cancel()
	report, err := a.model.Synthesize(callCtx, state)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(report) == "" || len(report) > 24*1024 {
		return nil, errors.New("model returned an empty or oversized research report")
	}
	return report, nil
}

func validateAnalysis(value analysis) error {
	if value.RelevanceScore < 1 || value.RelevanceScore > 10 || strings.TrimSpace(value.Summary) == "" || len(value.Summary) > 3000 {
		return errors.New("model returned invalid summary or relevance score")
	}
	for _, values := range [][]string{value.Insights, value.KeyPoints, value.ResearchGaps} {
		if values == nil || len(values) > 8 {
			return errors.New("model analysis arrays must contain at most eight items")
		}
		for _, text := range values {
			if len(text) > 1000 || strings.TrimSpace(text) == "" {
				return errors.New("model returned empty or oversized analysis text")
			}
		}
	}
	return nil
}

func fixturePaper(id string) (paper, error) {
	titles := map[string]string{
		"fixture-001": "Checkpointed workflows",
		"fixture-002": "Idempotent work execution",
		"fixture-003": "Recovery experiments",
	}
	title, exists := titles[id]
	if !exists {
		return paper{}, errors.New("unknown fixture paper ID")
	}
	return paper{
		ID: id, Title: title, Summary: "Synthetic fixture about " + strings.ToLower(title) + "; not a real academic paper.",
		Authors: []string{"Fictional Sample Author"}, Published: "2000-01-01",
		Categories: []string{"fixture"}, PrimaryCategory: "fixture", Source: "fixture",
	}, nil
}

func fixtureReport(state researchState) string {
	var report strings.Builder
	fmt.Fprintf(&report, "# Fixture research report\n\n"+
		"> Synthetic fixture only: no arXiv search or model inference was performed.\n\n"+
		"## Summary\nTopic: %s\nCompleted %d iterations with %d query analyses and %d synthetic papers.\n\n"+
		"## Key Findings\n", state.Topic, state.Iteration, len(state.Findings), len(state.Papers))
	for _, finding := range state.Findings {
		fmt.Fprintf(&report, "- %s: %s\n", finding.Query, finding.Summary)
	}
	report.WriteString("\n## Methods & Approaches\nDeterministic replay; idempotent activities; failure-injection tests (synthetic examples).\n\n" +
		"## Open Questions\nValidate all synthetic claims against real papers before academic use.\n\n## References\n")
	for _, paper := range state.Papers {
		fmt.Fprintf(&report, "- %s: %s (synthetic; not an arXiv paper).\n", paper.ID, paper.Title)
	}
	return report.String()
}
