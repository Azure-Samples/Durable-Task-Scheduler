package main

import (
	"encoding/json"
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
	demoTopic      = "durable workflow reliability"
	maxPapers      = 60
)

type paper struct {
	ID              string   `json:"arxiv_id"`
	Title           string   `json:"title"`
	Summary         string   `json:"summary"`
	Authors         []string `json:"authors"`
	Published       string   `json:"published"`
	Updated         string   `json:"updated,omitempty"`
	Categories      []string `json:"categories"`
	PrimaryCategory string   `json:"primary_category"`
	AbsURL          string   `json:"abs_url,omitempty"`
	PDFURL          string   `json:"pdf_url,omitempty"`
	Comment         string   `json:"comment,omitempty"`
	JournalRef      string   `json:"journal_ref,omitempty"`
	DOI             string   `json:"doi,omitempty"`
	Source          string   `json:"source"`
}

type analysis struct {
	Insights       []string `json:"insights"`
	RelevanceScore int      `json:"relevance_score"`
	Summary        string   `json:"summary"`
	KeyPoints      []string `json:"key_points"`
	ResearchGaps   []string `json:"research_gaps"`
}

type finding struct {
	Query string `json:"query"`
	analysis
	PaperIDs []string `json:"paper_ids"`
}

type researchState struct {
	Topic         string    `json:"topic"`
	Mode          string    `json:"mode"`
	MaxIterations int       `json:"max_iterations"`
	Iteration     int       `json:"current_iteration"`
	Queries       []string  `json:"queries"`
	Findings      []finding `json:"all_findings"`
	Papers        []paper   `json:"papers"`
}

type researchResult struct {
	Topic         string    `json:"topic"`
	Mode          string    `json:"mode"`
	Iterations    int       `json:"iterations"`
	FindingsCount int       `json:"findings_count"`
	PaperIDs      []string  `json:"paper_ids"`
	Report        string    `json:"report"`
	Findings      []finding `json:"findings"`
	Papers        []paper   `json:"papers"`
}

type progress struct {
	Mode          string   `json:"mode"`
	Phase         string   `json:"phase"`
	Iteration     int      `json:"iteration"`
	FindingsCount int      `json:"findings_count"`
	PaperIDs      []string `json:"paper_ids"`
}

type queryInput struct {
	Topic     string `json:"topic"`
	Query     string `json:"query"`
	Mode      string `json:"mode"`
	Iteration int    `json:"iteration"`
	Slot      int    `json:"slot"`
}

type queryResult struct {
	Finding finding `json:"finding"`
	Papers  []paper `json:"papers"`
}

type fetchInput struct {
	Mode string `json:"mode"`
	ID   string `json:"arxiv_id"`
}

type analysisInput struct {
	Mode   string  `json:"mode"`
	Topic  string  `json:"topic"`
	Query  string  `json:"query"`
	Papers []paper `json:"papers"`
}

func validateTopic(topic string) error {
	if strings.TrimSpace(topic) == "" || len(topic) > 200 || strings.ContainsAny(topic, "\x00\r\n") {
		return errors.New("topic must contain 1–200 bytes of non-blank, single-line text")
	}
	return nil
}

func validateQuery(query string) error {
	if strings.TrimSpace(query) == "" || len(query) > 300 || strings.ContainsAny(query, "\x00\r\n") {
		return errors.New("query must contain 1–300 bytes of non-blank, single-line text")
	}
	return nil
}

func validateState(state researchState) error {
	if err := validateTopic(state.Topic); err != nil {
		return err
	}
	if state.Mode != "fixture" && state.Mode != "real" {
		return errors.New("research mode must be fixture or real")
	}
	if state.MaxIterations < 1 || state.MaxIterations > 10 || state.Iteration < 0 || state.Iteration > state.MaxIterations {
		return errors.New("invalid research iteration budget")
	}
	if len(state.Queries) < 1 || len(state.Queries) > 2 {
		return errors.New("each iteration needs one or two research queries")
	}
	for _, query := range state.Queries {
		if err := validateQuery(query); err != nil {
			return err
		}
	}
	if len(state.Papers) > maxPapers || len(state.Findings) > 20 {
		return errors.New("research state exceeded its paper/finding budget")
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if len(data) > 512*1024 {
		return errors.New("research checkpoint exceeded 512 KiB")
	}
	return nil
}

func validateResult(result researchResult) error {
	if err := validateTopic(result.Topic); err != nil {
		return err
	}
	if (result.Mode != "fixture" && result.Mode != "real") || result.Iterations < 1 || result.Iterations > 10 ||
		result.FindingsCount < 1 || result.FindingsCount != len(result.Findings) || len(result.Papers) > maxPapers ||
		len(result.PaperIDs) != len(result.Papers) || strings.TrimSpace(result.Report) == "" || len(result.Report) > 24*1024 {
		return errors.New("invalid completed research fields")
	}
	for index, item := range result.Papers {
		if result.PaperIDs[index] != item.ID || item.ID == "" {
			return errors.New("completed paper IDs do not match fetched evidence")
		}
	}
	return nil
}

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
