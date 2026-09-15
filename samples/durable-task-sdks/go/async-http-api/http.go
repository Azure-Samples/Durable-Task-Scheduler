package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
)

type operationStore interface {
	Start(context.Context, operationInput) (api.InstanceID, error)
	Get(context.Context, api.InstanceID) (*api.OrchestrationMetadata, error)
	Terminate(context.Context, api.InstanceID) error
}

type schedulerStore struct{ client *dts.Client }

func (s schedulerStore) Start(ctx context.Context, input operationInput) (api.InstanceID, error) {
	return s.client.ScheduleNewOrchestration(ctx, orchestratorName,
		api.WithInstanceID(api.InstanceID(input.OperationID)), api.WithInput(input))
}

func (s schedulerStore) Get(ctx context.Context, id api.InstanceID) (*api.OrchestrationMetadata, error) {
	return s.client.FetchOrchestrationMetadata(ctx, id, api.WithFetchPayloads(true))
}

func (s schedulerStore) Terminate(ctx context.Context, id api.InstanceID) error {
	return s.client.TerminateOrchestration(ctx, id, api.WithOutput("Terminated by HTTP client"))
}

type startResponse struct {
	OperationID string `json:"operation_id"`
	StatusURL   string `json:"status_url"`
}

type statusResponse struct {
	OperationID string           `json:"operation_id"`
	Status      string           `json:"status"`
	LastUpdated time.Time        `json:"last_updated"`
	Result      *operationResult `json:"result,omitempty"`
	Error       string           `json:"error,omitempty"`
}

var operationIDPattern = regexp.MustCompile(`^go-async-http-[a-z0-9-]{1,100}$`)

func newHandler(store operationStore) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/start-operation", func(w http.ResponseWriter, r *http.Request) {
		defaultTime := 5
		input := &struct {
			ProcessingTime *int `json:"processing_time"`
		}{ProcessingTime: &defaultTime}
		if !readJSON(w, r, &input) {
			return
		}
		if input == nil || input.ProcessingTime == nil {
			writeError(w, http.StatusBadRequest, "expected a JSON object with a non-null processing_time")
			return
		}
		if err := validateProcessingTime(*input.ProcessingTime); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		id := sample.ID("async-http")
		actualID, err := store.Start(r.Context(), operationInput{string(id), *input.ProcessingTime})
		if err != nil {
			backendError(w, err)
			return
		}
		if actualID != id {
			writeError(w, http.StatusBadGateway, "scheduler returned an unexpected operation ID")
			return
		}
		location := "/api/operations/" + string(id)
		setPollingHeaders(w, location)
		writeJSON(w, http.StatusAccepted, startResponse{string(id), location})
	})
	get := func(w http.ResponseWriter, r *http.Request) *api.OrchestrationMetadata {
		id := r.PathValue("id")
		if !operationIDPattern.MatchString(id) {
			writeError(w, http.StatusNotFound, "operation not found")
			return nil
		}
		metadata, err := store.Get(r.Context(), api.InstanceID(id))
		if err != nil {
			backendError(w, err)
			return nil
		}
		if metadata == nil || metadata.Name != orchestratorName {
			writeError(w, http.StatusNotFound, "operation not found")
			return nil
		}
		return metadata
	}
	mux.HandleFunc("GET /api/operations/{id}", func(w http.ResponseWriter, r *http.Request) {
		metadata := get(w, r)
		if metadata == nil {
			return
		}
		response := statusResponse{
			OperationID: string(metadata.InstanceID), Status: statusName(metadata.RuntimeStatus),
			LastUpdated: metadata.LastUpdatedAt,
		}
		code := http.StatusOK
		switch metadata.RuntimeStatus {
		case api.RUNTIME_STATUS_COMPLETED:
			response.Result = &operationResult{}
			if err := metadata.ReadOutput(response.Result); err != nil {
				writeError(w, http.StatusBadGateway, "invalid durable result")
				return
			}
			if response.Result.OperationID != string(metadata.InstanceID) || response.Result.OperationID == "" ||
				response.Result.Status != "completed" || response.Result.ProcessedAt <= 0 ||
				response.Result.Result != fmt.Sprintf("Operation %s completed successfully", metadata.InstanceID) {
				writeError(w, http.StatusBadGateway, "invalid durable result fields")
				return
			}
		case api.RUNTIME_STATUS_FAILED:
			response.Error = "operation failed; inspect the DTS dashboard for details"
		case api.RUNTIME_STATUS_TERMINATED, api.RUNTIME_STATUS_CANCELED:
		default:
			code = http.StatusAccepted
			setPollingHeaders(w, "/api/operations/"+string(metadata.InstanceID))
		}
		writeJSON(w, code, response)
	})
	mux.HandleFunc("DELETE /api/operations/{id}", func(w http.ResponseWriter, r *http.Request) {
		metadata := get(w, r)
		if metadata == nil {
			return
		}
		switch metadata.RuntimeStatus {
		case api.RUNTIME_STATUS_COMPLETED, api.RUNTIME_STATUS_FAILED, api.RUNTIME_STATUS_TERMINATED, api.RUNTIME_STATUS_CANCELED:
			writeError(w, http.StatusConflict, "operation is already terminal")
			return
		}
		if err := store.Terminate(r.Context(), metadata.InstanceID); err != nil {
			backendError(w, err)
			return
		}
		location := "/api/operations/" + string(metadata.InstanceID)
		setPollingHeaders(w, location)
		writeJSON(w, http.StatusAccepted, startResponse{string(metadata.InstanceID), location})
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		w.Header().Set("Cache-Control", "no-store")
		mux.ServeHTTP(w, r.WithContext(ctx))
	})
}

func statusName(status api.OrchestrationStatus) string {
	name := strings.TrimPrefix(status.String(), "ORCHESTRATION_STATUS_")
	if name == "" {
		return "Unknown"
	}
	return name[:1] + strings.ToLower(name[1:])
}

func setPollingHeaders(w http.ResponseWriter, location string) {
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
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
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
		writeError(w, http.StatusNotFound, "operation not found")
	case errors.Is(err, context.DeadlineExceeded):
		writeError(w, http.StatusGatewayTimeout, "scheduler request timed out")
	case errors.Is(err, context.Canceled):
		writeError(w, http.StatusRequestTimeout, "request canceled")
	default:
		writeError(w, http.StatusBadGateway, "scheduler request failed")
	}
}

func loopbackAddress(address string) (string, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", fmt.Errorf("listen address must be a loopback host:port: %w", err)
	}
	if strings.EqualFold(host, "localhost") {
		host = "127.0.0.1"
	}
	if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
		return "", errors.New("listen address must use a loopback IP or localhost")
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 0 || number > 65535 {
		return "", errors.New("invalid listen port")
	}
	return net.JoinHostPort(host, port), nil
}

func serveHTTP(ctx context.Context, address string, handler http.Handler) error {
	address, err := loopbackAddress(address)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	server := &http.Server{
		Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second,
		WriteTimeout: 15 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 * 1024,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}
	result := make(chan error, 1)
	go func() { result <- server.Serve(listener) }()
	fmt.Printf("Async HTTP API listening on http://%s (until -timeout or Ctrl+C)\n", listener.Addr())
	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := server.Shutdown(shutdown)
		if err != nil {
			err = errors.Join(err, server.Close())
		}
		serveErr := <-result
		if errors.Is(serveErr, http.ErrServerClosed) {
			serveErr = nil
		}
		return errors.Join(err, serveErr)
	}
}
