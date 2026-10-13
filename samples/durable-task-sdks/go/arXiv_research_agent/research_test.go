package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/microsoft/durabletask-go/api"
)

type activityContext struct {
	input any
	ctx   context.Context
}

func (c activityContext) GetInput(target any) error {
	data, err := json.Marshal(c.input)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
func (c activityContext) Context() context.Context {
	if c.ctx != nil {
		return c.ctx
	}
	return context.Background()
}

func fixtureResult(t *testing.T) (researchResult, researchState) {
	t.Helper()
	a := &activities{mode: "fixture"}
	state := researchState{Topic: demoTopic, Mode: "fixture", MaxIterations: 2, Queries: []string{demoTopic}, Findings: []finding{}, Papers: []paper{}}
	var checkpoint researchState
	for iteration := 1; iteration <= 2; iteration++ {
		state.Iteration = iteration
		for slot, query := range state.Queries {
			idsValue, err := a.search(activityContext{input: queryInput{state.Topic, query, "fixture", iteration, slot}})
			if err != nil {
				t.Fatal(err)
			}
			ids := idsValue.([]string)
			papers := make([]paper, 0, len(ids))
			for _, id := range ids {
				value, err := a.fetch(activityContext{input: fetchInput{"fixture", id}})
				if err != nil {
					t.Fatal(err)
				}
				papers = append(papers, value.(paper))
			}
			value, err := a.analyze(activityContext{input: analysisInput{"fixture", state.Topic, query, papers}})
			if err != nil {
				t.Fatal(err)
			}
			state.Findings = append(state.Findings, value.(finding))
			state.Papers = mergePapers(state.Papers, papers)
		}
		decision, err := a.decide(activityContext{input: state})
		if err != nil || decision.(bool) != (iteration < 2) {
			t.Fatalf("bad continuation decision: %v %v", decision, err)
		}
		if decision.(bool) {
			value, err := a.gaps(activityContext{input: state})
			if err != nil {
				t.Fatal(err)
			}
			state.Queries = freshQueries(value.([]string), state.Findings)
			data, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(data, &checkpoint); err != nil {
				t.Fatal(err)
			}
			if err := validateState(checkpoint); err != nil {
				t.Fatal(err)
			}
		}
	}
	value, err := a.synthesize(activityContext{input: state})
	if err != nil {
		t.Fatal(err)
	}
	result := researchResult{
		Topic: state.Topic, Mode: state.Mode, Iterations: state.Iteration, FindingsCount: len(state.Findings),
		PaperIDs: paperIDs(state.Papers), Report: value.(string), Findings: state.Findings, Papers: state.Papers,
	}
	return result, checkpoint
}

func TestFixtureActivitiesAndCheckpoint(t *testing.T) {
	result, checkpoint := fixtureResult(t)
	if err := verifyFixtureResult(result); err != nil {
		t.Fatal(err)
	}
	if checkpoint.Iteration != 1 || len(checkpoint.Findings) != 1 || len(checkpoint.Queries) != 2 {
		t.Fatalf("checkpoint missing accumulated state: %+v", checkpoint)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip researchResult
	if err := json.Unmarshal(encoded, &roundTrip); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result, roundTrip) {
		t.Fatalf("result JSON loses analysis fields: %s", encoded)
	}
	if _, err := fixturePaper("2301.12345"); err == nil {
		t.Fatal("fixture accepted a real-looking paper ID")
	}
	a := &activities{mode: "fixture"}
	if _, err := a.search(activityContext{input: queryInput{Mode: "real", Query: "test"}}); err == nil {
		t.Fatal("real request silently fell back to fixtures")
	}
	if _, err := newRegistry(a); err != nil {
		t.Fatal(err)
	}
}

func TestQueryAndStateBudgets(t *testing.T) {
	state := researchState{Topic: demoTopic, Mode: "fixture", MaxIterations: 2, Queries: []string{demoTopic}}
	if err := validateState(state); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*researchState){
		func(s *researchState) { s.Topic = "" },
		func(s *researchState) { s.MaxIterations = 11 },
		func(s *researchState) { s.Iteration = 3 },
		func(s *researchState) { s.Mode = "auto" },
		func(s *researchState) { s.Queries = nil },
		func(s *researchState) { s.Queries = []string{"one", "two", "three"} },
		func(s *researchState) { s.Queries = []string{strings.Repeat("x", 301)} },
		func(s *researchState) { s.Papers = make([]paper, maxPapers+1) },
	} {
		copy := state
		change(&copy)
		if err := validateState(copy); err == nil {
			t.Fatalf("invalid checkpoint accepted: %+v", copy)
		}
	}
	actual := freshQueries([]string{" Existing ", "new", "NEW", "other", "third"}, []finding{{Query: "existing"}})
	if !reflect.DeepEqual(actual, []string{"new", "other"}) {
		t.Fatalf("fresh query selection=%v", actual)
	}
	papers := mergePapers([]paper{{ID: "b"}, {ID: "a"}}, []paper{{ID: "a"}, {ID: "c"}})
	if !reflect.DeepEqual(paperIDs(papers), []string{"a", "b", "c"}) {
		t.Fatal("paper merge is not deterministic and deduplicated")
	}
}

type fakeStore struct {
	metadata   *api.OrchestrationMetadata
	waitResult *api.OrchestrationMetadata
	query      *api.OrchestrationQueryResult
	err        error
	waitErr    error
	started    researchState
	delay      int
	terminated bool
	token      string
}

func (s *fakeStore) Start(_ context.Context, id api.InstanceID, state researchState, delay int) (api.InstanceID, error) {
	s.started, s.delay = state, delay
	return id, s.err
}
func (s *fakeStore) Get(context.Context, api.InstanceID) (*api.OrchestrationMetadata, error) {
	return s.metadata, s.err
}
func (s *fakeStore) Wait(context.Context, api.InstanceID) (*api.OrchestrationMetadata, error) {
	return s.waitResult, s.waitErr
}
func (s *fakeStore) Terminate(context.Context, api.InstanceID) error {
	s.terminated = true
	return s.err
}
func (s *fakeStore) List(_ context.Context, token string) (*api.OrchestrationQueryResult, error) {
	s.token = token
	return s.query, s.err
}

func metadataFor(t *testing.T, status api.OrchestrationStatus) *api.OrchestrationMetadata {
	t.Helper()
	result, _ := fixtureResult(t)
	input, err := json.Marshal(researchState{
		Topic: demoTopic, Mode: "fixture", MaxIterations: 2,
		Queries: []string{demoTopic}, Findings: []finding{}, Papers: []paper{},
	})
	if err != nil {
		t.Fatal(err)
	}
	output, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	return &api.OrchestrationMetadata{
		InstanceID: "go-arxiv-test", ExecutionID: "execution-2", Name: researchName, RuntimeStatus: status,
		CreatedAt:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		SerializedInput: string(input), SerializedOutput: string(output),
	}
}

func TestHTTPStartValidation(t *testing.T) {
	for _, test := range []struct {
		body, media string
		code        int
	}{
		{`{"topic":"  durable workflow reliability  "}`, "application/json", 202},
		{`{"topic":"test","max_iterations":2,"start_delay_seconds":30}`, "application/json", 202},
		{`{}`, "application/json", 400},
		{`null`, "application/json", 400},
		{`{"topic":"test","max_iterations":0}`, "application/json", 400},
		{`{"topic":"test","max_iterations":11}`, "application/json", 400},
		{`{"topic":"test","max_iterations":"2"}`, "application/json", 400},
		{`{"topic":"test","max_iterations":null}`, "application/json", 400},
		{`{"topic":"test","start_delay_seconds":null}`, "application/json", 400},
		{`{"topic":"test","max_iterations":1.5}`, "application/json", 400},
		{`{"topic":"test","start_delay_seconds":31}`, "application/json", 400},
		{`{"topic":"test","mode":"real"}`, "application/json", 400},
		{`{"topic":"test"} {}`, "application/json", 400},
		{`{"topic":"test"}`, "text/plain", 415},
		{strings.Repeat(" ", 4097) + `{}`, "application/json", 413},
	} {
		t.Run(test.body[:min(len(test.body), 40)], func(t *testing.T) {
			store := &fakeStore{}
			server := httptest.NewServer((&researchAPI{store: store, mode: "fixture"}).handler())
			defer server.Close()
			response, err := server.Client().Post(server.URL+"/agents", test.media, strings.NewReader(test.body))
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode != test.code {
				t.Fatalf("status=%d want=%d", response.StatusCode, test.code)
			}
			if test.code == 202 {
				var started startResponse
				if err := json.NewDecoder(response.Body).Decode(&started); err != nil {
					t.Fatal(err)
				}
				if !started.OK || !instancePattern.MatchString(started.InstanceID) ||
					response.Header.Get("Location") != started.StatusURL || response.Header.Get("Retry-After") != "1" ||
					store.started.Mode != "fixture" || store.started.Iteration != 0 {
					t.Fatalf("invalid accepted research: %+v %+v", started, store.started)
				}
			} else if store.started.Topic != "" {
				t.Fatal("invalid input scheduled research")
			}
		})
	}
}

func TestHTTPStatusWaitListAndTerminate(t *testing.T) {
	completed := metadataFor(t, api.RUNTIME_STATUS_COMPLETED)
	store := &fakeStore{metadata: completed, waitResult: completed, query: &api.OrchestrationQueryResult{
		Orchestrations: []*api.OrchestrationMetadata{completed, {Name: paperName}}, ContinuationToken: "next",
	}}
	server := httptest.NewServer((&researchAPI{store: store, mode: "fixture"}).handler())
	defer server.Close()
	ctx := context.Background()
	var status statusResponse
	code, _, err := requestJSON(ctx, server.Client(), http.MethodGet, server.URL+"/agents/go-arxiv-test", nil, &status)
	if err != nil || code != 200 || status.Status != "COMPLETED" || status.Mode != "fixture" ||
		status.Report != expectedDemoReport || status.Iteration != 2 || status.FindingsCount != 3 {
		t.Fatalf("status response: %d %+v %v", code, status, err)
	}
	var result researchResult
	code, _, err = requestJSON(ctx, server.Client(), http.MethodGet, server.URL+"/agents/go-arxiv-test/wait?timeout=5", nil, &result)
	if err != nil || code != 200 {
		t.Fatalf("wait failed: %d %v", code, err)
	}
	if err := verifyFixtureResult(result); err != nil {
		t.Fatal(err)
	}
	var list struct {
		Agents []statusResponse `json:"agents"`
		Token  string           `json:"continuation_token"`
	}
	code, _, err = requestJSON(ctx, server.Client(), http.MethodGet, server.URL+"/agents?continuation_token=previous", nil, &list)
	if err != nil || code != 200 || list.Token != "next" || len(list.Agents) != 1 || store.token != "previous" {
		t.Fatalf("invalid paged list: %d %+v %v", code, list, err)
	}
	code, _, err = requestJSON(ctx, server.Client(), http.MethodDelete, server.URL+"/agents/go-arxiv-test", nil, nil)
	if err != nil || code != 409 || store.terminated {
		t.Fatal("terminal research was terminated again")
	}
}

func TestHTTPPendingTerminationAndFailures(t *testing.T) {
	for _, test := range []struct {
		name, method, path string
		status             api.OrchestrationStatus
		err, waitErr       error
		code               int
	}{
		{"pending", "GET", "/agents/go-arxiv-test", api.RUNTIME_STATUS_PENDING, nil, nil, 200},
		{"terminate", "DELETE", "/agents/go-arxiv-test", api.RUNTIME_STATUS_RUNNING, nil, nil, 202},
		{"missing", "GET", "/agents/go-arxiv-test", api.RUNTIME_STATUS_RUNNING, api.ErrInstanceNotFound, nil, 404},
		{"backend", "GET", "/agents/go-arxiv-test", api.RUNTIME_STATUS_RUNNING, errors.New("secret error"), nil, 502},
		{"timeout", "GET", "/agents/go-arxiv-test/wait", api.RUNTIME_STATUS_RUNNING, nil, context.DeadlineExceeded, 408},
		{"failed", "GET", "/agents/go-arxiv-test/wait", api.RUNTIME_STATUS_FAILED, nil, nil, 500},
		{"canceled", "GET", "/agents/go-arxiv-test/wait", api.RUNTIME_STATUS_TERMINATED, nil, nil, 409},
		{"bad timeout", "GET", "/agents/go-arxiv-test/wait?timeout=0", api.RUNTIME_STATUS_RUNNING, nil, nil, 400},
		{"long timeout", "GET", "/agents/go-arxiv-test/wait?timeout=61", api.RUNTIME_STATUS_RUNNING, nil, nil, 400},
		{"bad ID", "GET", "/agents/@other@id", api.RUNTIME_STATUS_RUNNING, nil, nil, 404},
		{"unsupported list", "GET", "/agents", api.RUNTIME_STATUS_RUNNING, api.ErrFeatureNotSupported, nil, 501},
	} {
		t.Run(test.name, func(t *testing.T) {
			metadata := metadataFor(t, test.status)
			store := &fakeStore{metadata: metadata, waitResult: metadata, err: test.err, waitErr: test.waitErr}
			w := httptest.NewRecorder()
			(&researchAPI{store: store, mode: "fixture"}).handler().ServeHTTP(w, httptest.NewRequest(test.method, test.path, nil))
			if w.Code != test.code || strings.Contains(w.Body.String(), "secret") {
				t.Fatalf("response=%d expected=%d %s", w.Code, test.code, w.Body.String())
			}
			if test.name == "terminate" && (!store.terminated || w.Header().Get("Retry-After") != "1") {
				t.Fatal("termination was not scheduled")
			}
		})
	}
	foreign := metadataFor(t, api.RUNTIME_STATUS_RUNNING)
	foreign.Name = "OtherGoSample"
	store := &fakeStore{metadata: foreign}
	w := httptest.NewRecorder()
	(&researchAPI{store: store}).handler().ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/agents/go-arxiv-test", nil))
	if w.Code != 404 || store.terminated {
		t.Fatal("API allowed termination of a different sample")
	}
	for _, address := range []string{":8000", "0.0.0.0:8000", "[::]:8000", "example.com:8000", "localhost:65536"} {
		if _, err := loopbackAddress(address); err == nil {
			t.Fatalf("unsafe listen address accepted: %s", address)
		}
	}
}

func TestFixtureResultAssertionsRejectWrongData(t *testing.T) {
	result, _ := fixtureResult(t)
	for _, change := range []func(*researchResult){
		func(r *researchResult) { r.Iterations = 1 },
		func(r *researchResult) { r.Report = "placeholder report" },
		func(r *researchResult) { r.Mode = "real" },
		func(r *researchResult) { r.FindingsCount = 0 },
		func(r *researchResult) { r.PaperIDs = []string{"2301.12345"} },
	} {
		copy := result
		change(&copy)
		if err := verifyFixtureResult(copy); err == nil {
			t.Fatal("strict demo assertions accepted altered output")
		}
	}
	for _, value := range []string{"", "null", `{}`, `{"should_continue":"yes"}`} {
		var target struct {
			Continue *bool `json:"should_continue"`
		}
		err := decodeJSON(value, &target)
		if err == nil && target.Continue != nil {
			t.Fatal(fmt.Sprintf("invalid decision decoded: %s", value))
		}
	}
}
