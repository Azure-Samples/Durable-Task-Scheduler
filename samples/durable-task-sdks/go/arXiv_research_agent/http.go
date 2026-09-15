package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
)

type researchStore interface {
	Start(context.Context, api.InstanceID, researchState, int) (api.InstanceID, error)
	Get(context.Context, api.InstanceID) (*api.OrchestrationMetadata, error)
	Wait(context.Context, api.InstanceID) (*api.OrchestrationMetadata, error)
	Terminate(context.Context, api.InstanceID) error
	List(context.Context, string) (*api.OrchestrationQueryResult, error)
}

type schedulerStore struct{ client *dts.Client }

func (s schedulerStore) Start(ctx context.Context, id api.InstanceID, state researchState, delay int) (api.InstanceID, error) {
	options := []api.NewOrchestrationOptions{api.WithInstanceID(id), api.WithInput(state)}
	if delay > 0 {
		options = append(options, api.WithStartTime(time.Now().Add(time.Duration(delay)*time.Second)))
	}
	return s.client.ScheduleNewOrchestration(ctx, researchName, options...)
}

func (s schedulerStore) Get(ctx context.Context, id api.InstanceID) (*api.OrchestrationMetadata, error) {
	return s.client.FetchOrchestrationMetadata(ctx, id, api.WithFetchPayloads(true))
}

func (s schedulerStore) Wait(ctx context.Context, id api.InstanceID) (*api.OrchestrationMetadata, error) {
	return s.client.WaitForOrchestrationCompletion(ctx, id, api.WithFetchPayloads(true))
}

func (s schedulerStore) Terminate(ctx context.Context, id api.InstanceID) error {
	return s.client.TerminateOrchestration(ctx, id, api.WithRecursiveTerminate(true), api.WithOutput("Terminated by HTTP client"))
}

func (s schedulerStore) List(ctx context.Context, token string) (*api.OrchestrationQueryResult, error) {
	return s.client.QueryInstances(ctx, api.OrchestrationQuery{
		InstanceIDPrefix: "go-arxiv-", PageSize: 20, ContinuationToken: token, FetchInputsAndOutputs: true,
	})
}

type researchAPI struct {
	store researchStore
	mode  string
}

var instancePattern = regexp.MustCompile(`^go-arxiv-[a-z0-9-]{1,100}$`)

func (s *researchAPI) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "healthy", "mode": s.mode})
	})
	mux.HandleFunc("POST /agents", s.start)
	mux.HandleFunc("GET /agents", s.list)
	mux.HandleFunc("GET /agents/{id}", s.status)
	mux.HandleFunc("GET /agents/{id}/wait", s.wait)
	mux.HandleFunc("DELETE /agents/{id}", s.terminate)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 65*time.Second)
		defer cancel()
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Research-Mode", s.mode)
		mux.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *researchAPI) start(w http.ResponseWriter, r *http.Request) {
	defaultIterations, defaultDelay := 3, 0
	body := &struct {
		Topic             string `json:"topic"`
		MaxIterations     *int   `json:"max_iterations"`
		StartDelaySeconds *int   `json:"start_delay_seconds"`
	}{MaxIterations: &defaultIterations, StartDelaySeconds: &defaultDelay}
	if !readJSON(w, r, &body) {
		return
	}
	if body == nil || body.MaxIterations == nil || body.StartDelaySeconds == nil {
		writeError(w, http.StatusBadRequest, "expected a JSON object with non-null iteration and delay fields")
		return
	}
	input := startRequest{Topic: strings.TrimSpace(body.Topic), MaxIterations: *body.MaxIterations, StartDelaySeconds: *body.StartDelaySeconds}
	if err := validateTopic(input.Topic); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if input.MaxIterations < 1 || input.MaxIterations > 10 || input.StartDelaySeconds < 0 || input.StartDelaySeconds > 30 {
		writeError(w, http.StatusBadRequest, "max_iterations must be 1–10 and start_delay_seconds must be 0–30")
		return
	}
	state := researchState{
		Topic: input.Topic, Mode: s.mode, MaxIterations: input.MaxIterations,
		Queries: []string{input.Topic}, Findings: []finding{}, Papers: []paper{},
	}
	id := sample.ID("arxiv")
	callCtx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	actualID, err := s.store.Start(callCtx, id, state, input.StartDelaySeconds)
	if err != nil {
		backendError(w, err)
		return
	}
	if actualID != id {
		writeError(w, http.StatusBadGateway, "scheduler returned an unexpected instance ID")
		return
	}
	location := "/agents/" + string(id)
	pollingHeaders(w, location)
	writeJSON(w, http.StatusAccepted, startResponse{true, string(id), location, s.mode})
}

func (s *researchAPI) metadata(w http.ResponseWriter, r *http.Request) *api.OrchestrationMetadata {
	id := r.PathValue("id")
	if !instancePattern.MatchString(id) {
		writeError(w, http.StatusNotFound, "research agent not found")
		return nil
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	metadata, err := s.store.Get(ctx, api.InstanceID(id))
	if err != nil {
		backendError(w, err)
		return nil
	}
	if metadata == nil || metadata.Name != researchName {
		writeError(w, http.StatusNotFound, "research agent not found")
		return nil
	}
	return metadata
}

func (s *researchAPI) status(w http.ResponseWriter, r *http.Request) {
	metadata := s.metadata(w, r)
	if metadata == nil {
		return
	}
	response, err := researchStatus(metadata)
	if err != nil {
		writeError(w, http.StatusBadGateway, "invalid durable research state")
		return
	}
	if !terminal(metadata.RuntimeStatus) {
		pollingHeaders(w, "/agents/"+string(metadata.InstanceID))
	}
	writeJSON(w, http.StatusOK, response)
}

func researchStatus(metadata *api.OrchestrationMetadata) (statusResponse, error) {
	response := statusResponse{
		AgentID: string(metadata.InstanceID), Status: strings.TrimPrefix(metadata.RuntimeStatus.String(), "ORCHESTRATION_STATUS_"),
		CreatedAt: metadata.CreatedAt,
	}
	// Metadata input can remain the original request after ContinueAsNew;
	// current progress comes from custom status and, once completed, the output.
	var state researchState
	if err := metadata.ReadInput(&state); err != nil {
		return response, err
	}
	response.Topic, response.Mode = state.Topic, state.Mode
	response.progress = progress{Mode: state.Mode, Phase: "pending", Iteration: state.Iteration,
		FindingsCount: len(state.Findings), PaperIDs: paperIDs(state.Papers)}
	if metadata.SerializedCustomStatus != "" {
		if err := metadata.ReadCustomStatus(&response.progress); err != nil {
			return response, err
		}
	}
	if metadata.RuntimeStatus == api.RUNTIME_STATUS_COMPLETED {
		var result researchResult
		if err := metadata.ReadOutput(&result); err != nil {
			return response, err
		}
		if err := validateResult(result); err != nil {
			return response, err
		}
		response.Topic, response.Mode, response.Report = result.Topic, result.Mode, result.Report
		response.progress = progress{result.Mode, "completed", result.Iterations, result.FindingsCount, result.PaperIDs}
	}
	if metadata.RuntimeStatus == api.RUNTIME_STATUS_FAILED {
		response.Error = "research failed; inspect the DTS dashboard for activity failure details"
	}
	return response, nil
}

func (s *researchAPI) wait(w http.ResponseWriter, r *http.Request) {
	seconds := 30
	if value := r.URL.Query().Get("timeout"); value != "" {
		number, err := strconv.Atoi(value)
		if err != nil || number < 1 || number > 60 {
			writeError(w, http.StatusBadRequest, "timeout must be an integer from 1–60 seconds")
			return
		}
		seconds = number
	}
	metadata := s.metadata(w, r)
	if metadata == nil {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(seconds)*time.Second)
	defer cancel()
	result, err := s.store.Wait(ctx, metadata.InstanceID)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			writeError(w, http.StatusRequestTimeout, "timeout waiting for research; the durable job is still running")
		} else {
			backendError(w, err)
		}
		return
	}
	if result == nil {
		writeError(w, http.StatusBadGateway, "scheduler returned no completion metadata")
		return
	}
	switch result.RuntimeStatus {
	case api.RUNTIME_STATUS_FAILED:
		writeError(w, http.StatusInternalServerError, "research failed; inspect the DTS dashboard")
		return
	case api.RUNTIME_STATUS_TERMINATED, api.RUNTIME_STATUS_CANCELED:
		writeError(w, http.StatusConflict, "research was terminated or canceled")
		return
	case api.RUNTIME_STATUS_COMPLETED:
	default:
		writeError(w, http.StatusBadGateway, "scheduler returned a nonterminal completion")
		return
	}
	var output researchResult
	if err := result.ReadOutput(&output); err != nil {
		writeError(w, http.StatusBadGateway, "invalid durable research result")
		return
	}
	if err := validateResult(output); err != nil {
		writeError(w, http.StatusBadGateway, "invalid durable research result fields")
		return
	}
	writeJSON(w, http.StatusOK, output)
}

func (s *researchAPI) terminate(w http.ResponseWriter, r *http.Request) {
	metadata := s.metadata(w, r)
	if metadata == nil {
		return
	}
	if terminal(metadata.RuntimeStatus) {
		writeError(w, http.StatusConflict, "research is already terminal")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if err := s.store.Terminate(ctx, metadata.InstanceID); err != nil {
		backendError(w, err)
		return
	}
	pollingHeaders(w, "/agents/"+string(metadata.InstanceID))
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "message": "Termination requested, including child orchestrations."})
}

func (s *researchAPI) list(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("continuation_token")
	if len(token) > 4096 {
		writeError(w, http.StatusBadRequest, "continuation token is too large")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	results, err := s.store.List(ctx, token)
	if err != nil {
		backendError(w, err)
		return
	}
	if results == nil {
		writeError(w, http.StatusBadGateway, "scheduler returned no query result")
		return
	}
	agents := []statusResponse{}
	for _, metadata := range results.Orchestrations {
		if metadata == nil || metadata.Name != researchName {
			continue
		}
		item, err := researchStatus(metadata)
		if err != nil {
			writeError(w, http.StatusBadGateway, "invalid durable research state")
			return
		}
		agents = append(agents, item)
	}
	writeJSON(w, http.StatusOK, struct {
		Agents            []statusResponse `json:"agents"`
		ContinuationToken string           `json:"continuation_token,omitempty"`
	}{agents, results.ContinuationToken})
}

func terminal(status api.OrchestrationStatus) bool {
	return status == api.RUNTIME_STATUS_COMPLETED || status == api.RUNTIME_STATUS_FAILED ||
		status == api.RUNTIME_STATUS_TERMINATED || status == api.RUNTIME_STATUS_CANCELED
}

func pollingHeaders(w http.ResponseWriter, location string) {
	w.Header().Set("Location", location)
	w.Header().Set("Retry-After", "1")
}

func readJSON(w http.ResponseWriter, r *http.Request, input any) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(input)
	if err == nil {
		var extra any
		if next := decoder.Decode(&extra); next != io.EOF {
			err = errors.New("expected exactly one JSON object")
			if next != nil {
				err = next
			}
		}
	}
	if err != nil {
		var large *http.MaxBytesError
		if errors.As(err, &large) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body exceeds 4096 bytes")
		} else {
			writeError(w, http.StatusBadRequest, "invalid JSON request")
		}
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, map[string]string{"error": message})
}

func backendError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, api.ErrInstanceNotFound):
		writeError(w, http.StatusNotFound, "research agent not found")
	case errors.Is(err, api.ErrFeatureNotSupported):
		writeError(w, http.StatusNotImplemented, "scheduler does not support listing; query an instance ID or use the DTS dashboard")
	case errors.Is(err, context.DeadlineExceeded):
		writeError(w, http.StatusGatewayTimeout, "DTS request timed out")
	case errors.Is(err, context.Canceled):
		writeError(w, http.StatusRequestTimeout, "HTTP request canceled; durable work may still be running")
	default:
		writeError(w, http.StatusBadGateway, "DTS request failed")
	}
}
