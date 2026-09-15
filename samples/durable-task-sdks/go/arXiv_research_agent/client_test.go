package main

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"
)

func TestDemoStartsOneResearchJob(t *testing.T) {
	var starts, waits atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/agents":
			starts.Add(1)
			var input startRequest
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Error(err)
			}
			if input.Topic != demoTopic || input.MaxIterations != 2 {
				t.Errorf("unexpected example research request: %+v", input)
			}
			writeJSON(w, http.StatusAccepted, startResponse{OK: true, InstanceID: "go-arxiv-example", StatusURL: "/agents/go-arxiv-example", Mode: "fixture"})
		case r.Method == http.MethodGet && r.URL.Path == "/agents/go-arxiv-example/wait":
			waits.Add(1)
			if r.URL.Query().Get("timeout") != "60" {
				t.Error("example wait must have a bounded timeout")
			}
			writeJSON(w, http.StatusOK, researchResult{Topic: demoTopic, Mode: "fixture", Report: "An example response."})
		default:
			t.Errorf("demo performed an unexpected request: %s %s", r.Method, r.URL)
			writeError(w, http.StatusNotFound, "unexpected request")
		}
	})
	// This checks demo HTTP behavior, not a simulated durable backend.
	if err := demo(t.Context(), handler); err != nil {
		t.Fatal(err)
	}
	if starts.Load() != 1 || waits.Load() != 1 {
		t.Fatalf("demo is not a one-job example: starts=%d waits=%d", starts.Load(), waits.Load())
	}
}
