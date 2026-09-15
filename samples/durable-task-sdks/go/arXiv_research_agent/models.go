package main

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const maxPapers = 60

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

type startRequest struct {
	Topic             string `json:"topic"`
	MaxIterations     int    `json:"max_iterations"`
	StartDelaySeconds int    `json:"start_delay_seconds,omitempty"`
}

type startResponse struct {
	OK         bool   `json:"ok"`
	InstanceID string `json:"instance_id"`
	StatusURL  string `json:"status_url"`
	Mode       string `json:"mode"`
}

type statusResponse struct {
	AgentID   string    `json:"agent_id"`
	Topic     string    `json:"topic"`
	Mode      string    `json:"mode"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	progress
	Report string `json:"report,omitempty"`
	Error  string `json:"error,omitempty"`
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
