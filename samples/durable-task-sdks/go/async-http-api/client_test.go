package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
)

func TestDemoStartsOneOperation(t *testing.T) {
	var starts, polls atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/start-operation":
			starts.Add(1)
			setPollingHeaders(w, "/api/operations/go-async-http-example")
			writeJSON(w, http.StatusAccepted, startResponse{"go-async-http-example", "/api/operations/go-async-http-example"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/operations/go-async-http-example":
			polls.Add(1)
			writeJSON(w, http.StatusOK, statusResponse{
				Status: "Completed", Result: &operationResult{OperationID: "go-async-http-example", Status: "completed", Result: "example result"},
			})
		default:
			t.Errorf("demo performed an unexpected request: %s %s", r.Method, r.URL)
			writeError(w, http.StatusNotFound, "unexpected request")
		}
	})
	// This tests the example client's HTTP behavior, not durable execution.
	if err := demo(t.Context(), handler); err != nil {
		t.Fatal(err)
	}
	if starts.Load() != 1 || polls.Load() != 1 {
		t.Fatalf("demo is not a one-operation example: starts=%d polls=%d", starts.Load(), polls.Load())
	}
}

func TestHTTPServerLifecycle(t *testing.T) {
	server, err := startHTTPServer(t.Context(), "127.0.0.1:0", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ready")
	}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Error(err)
		}
	})
	response, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("server returned %d", response.StatusCode)
	}
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
}
