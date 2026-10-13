package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/microsoft/durabletask-go/api"
)

type fakeStore struct {
	input      operationInput
	metadata   *api.OrchestrationMetadata
	err        error
	terminated bool
}

func (s *fakeStore) Start(_ context.Context, input operationInput) (api.InstanceID, error) {
	s.input = input
	return api.InstanceID(input.OperationID), s.err
}
func (s *fakeStore) Get(context.Context, api.InstanceID) (*api.OrchestrationMetadata, error) {
	return s.metadata, s.err
}
func (s *fakeStore) Terminate(context.Context, api.InstanceID) error {
	s.terminated = true
	return s.err
}

func TestHTTPStartAndValidation(t *testing.T) {
	for _, test := range []struct {
		name, contentType, body string
		code                    int
	}{
		{"default", "application/json", `{}`, 202},
		{"typed", "application/json; charset=utf-8", `{"processing_time":1}`, 202},
		{"zero", "application/json", `{"processing_time":0}`, 400},
		{"negative", "application/json", `{"processing_time":-1}`, 400},
		{"too long", "application/json", `{"processing_time":31}`, 400},
		{"fraction", "application/json", `{"processing_time":1.5}`, 400},
		{"string", "application/json", `{"processing_time":"1"}`, 400},
		{"null", "application/json", `null`, 400},
		{"null time", "application/json", `{"processing_time":null}`, 400},
		{"unknown", "application/json", `{"operation_id":"spoofed"}`, 400},
		{"trailing", "application/json", `{} {}`, 400},
		{"invalid", "application/json", `{`, 400},
		{"media", "text/plain", `{}`, 415},
		{"large", "application/json", strings.Repeat(" ", 4097) + `{}`, 413},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &fakeStore{}
			server := httptest.NewServer(newHandler(store))
			defer server.Close()
			req, err := http.NewRequest(http.MethodPost, server.URL+"/api/start-operation", strings.NewReader(test.body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", test.contentType)
			resp, err := server.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != test.code {
				data, _ := io.ReadAll(resp.Body)
				t.Fatalf("status=%d want=%d body=%s", resp.StatusCode, test.code, data)
			}
			if test.code == http.StatusAccepted {
				var result startResponse
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatal(err)
				}
				if !operationIDPattern.MatchString(result.OperationID) || result.OperationID != store.input.OperationID ||
					resp.Header.Get("Location") != result.StatusURL || resp.Header.Get("Retry-After") != "1" {
					t.Fatalf("bad accepted response: %+v headers=%v", result, resp.Header)
				}
				if test.name == "default" && store.input.ProcessingTime != 5 {
					t.Fatalf("default processing time=%d", store.input.ProcessingTime)
				}
			} else if store.input.OperationID != "" {
				t.Fatal("invalid input scheduled work")
			}
		})
	}
}

func TestStatusAndTermination(t *testing.T) {
	id := api.InstanceID("go-async-http-test")
	for _, test := range []struct {
		name   string
		status api.OrchestrationStatus
		code   int
	}{
		{"Pending", api.RUNTIME_STATUS_PENDING, 202},
		{"Running", api.RUNTIME_STATUS_RUNNING, 202},
		{"Completed", api.RUNTIME_STATUS_COMPLETED, 200},
		{"Failed", api.RUNTIME_STATUS_FAILED, 200},
		{"Terminated", api.RUNTIME_STATUS_TERMINATED, 200},
		{"Canceled", api.RUNTIME_STATUS_CANCELED, 200},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &fakeStore{metadata: &api.OrchestrationMetadata{
				InstanceID: id, Name: orchestratorName, RuntimeStatus: test.status,
				SerializedOutput: `{"operation_id":"go-async-http-test","status":"completed","result":"Operation go-async-http-test completed successfully","processed_at":1}`,
			}}
			handler := newHandler(store)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/operations/"+string(id), nil))
			var response statusResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if w.Code != test.code || response.Status != test.name || response.OperationID != string(id) {
				t.Fatalf("unexpected result: %d %+v", w.Code, response)
			}
			if test.code == 202 && w.Header().Get("Retry-After") != "1" {
				t.Fatal("pending response lacks retry advice")
			}
			w = httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/operations/"+string(id), nil))
			if test.code == 202 && (!store.terminated || w.Code != 202) {
				t.Fatalf("termination not scheduled: %d", w.Code)
			}
			if test.code == 200 && (store.terminated || w.Code != 409) {
				t.Fatal("terminal work was terminated again")
			}
		})
	}
}

func TestBackendFailuresAndIsolation(t *testing.T) {
	for _, test := range []struct {
		name  string
		store fakeStore
		code  int
	}{
		{"absent", fakeStore{}, 404},
		{"missing", fakeStore{err: api.ErrInstanceNotFound}, 404},
		{"timeout", fakeStore{err: context.DeadlineExceeded}, 504},
		{"backend", fakeStore{err: errors.New("secret diagnostic")}, 502},
		{"foreign", fakeStore{metadata: &api.OrchestrationMetadata{Name: "OtherSample"}}, 404},
		{"bad output", fakeStore{metadata: &api.OrchestrationMetadata{
			Name: orchestratorName, RuntimeStatus: api.RUNTIME_STATUS_COMPLETED, SerializedOutput: "invalid",
		}}, 502},
		{"empty output", fakeStore{metadata: &api.OrchestrationMetadata{
			Name: orchestratorName, RuntimeStatus: api.RUNTIME_STATUS_COMPLETED, SerializedOutput: "{}",
		}}, 502},
	} {
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			newHandler(&test.store).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/operations/go-async-http-test", nil))
			if w.Code != test.code || strings.Contains(w.Body.String(), "secret") {
				t.Fatalf("response %d %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestActivityCancellationAndListenValidation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := processOperation(ctx, operationInput{ProcessingTime: 1}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	for _, address := range []string{":8000", "0.0.0.0:8000", "[::]:8000", "example.com:8000", "localhost:-1", "127.0.0.1:65536"} {
		if _, err := loopbackAddress(address); err == nil {
			t.Fatalf("accepted unsafe listen address %s", address)
		}
	}
	for _, address := range []string{"localhost:8000", "127.0.0.1:0", "[::1]:8000"} {
		if _, err := loopbackAddress(address); err != nil {
			t.Fatal(err)
		}
	}
}
