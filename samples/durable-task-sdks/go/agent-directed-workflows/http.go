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

type entityStore interface {
	Signal(context.Context, string, string, turnRequest) error
	State(context.Context, string) (*chatState, error)
}

type schedulerStore struct{ client *dts.Client }

func (s schedulerStore) Signal(ctx context.Context, session, operation string, input turnRequest) error {
	return s.client.SignalEntity(ctx, api.NewEntityID(entityName, session), operation, api.WithSignalInput(input))
}

func (s schedulerStore) State(ctx context.Context, session string) (*chatState, error) {
	metadata, err := s.client.GetEntity(ctx, api.NewEntityID(entityName, session))
	if err != nil || metadata == nil || !metadata.HasState {
		return nil, err
	}
	var state chatState
	if err := metadata.ReadState(&state); err != nil {
		return nil, err
	}
	return &state, nil
}

type chatAPI struct {
	store entityStore
	mode  string
	relay *streamRelay
}

type chatResponse struct {
	SessionID string `json:"sessionId"`
	Message   string `json:"message"`
	Mode      string `json:"mode"`
}

type historyResponse struct {
	SessionID string    `json:"sessionId"`
	History   []message `json:"history"`
	Mode      string    `json:"mode"`
}

type resetResponse struct {
	SessionID string `json:"sessionId"`
	Status    string `json:"status"`
}

type sseEvent struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
}

var sessionPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,80}$`)

func (s *chatAPI) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /chat/{session}", s.chat)
	mux.HandleFunc("GET /chat/{session}/history", s.history)
	mux.HandleFunc("POST /chat/{session}/reset", s.reset)
	mux.HandleFunc("GET /chat/{session}/requests/{request}", s.requestStatus)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 40*time.Second)
		defer cancel()
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Chat-Mode", s.mode)
		mux.ServeHTTP(w, r.WithContext(ctx))
	})
}

func validSession(w http.ResponseWriter, r *http.Request) bool {
	if !sessionPattern.MatchString(r.PathValue("session")) {
		writeError(w, http.StatusBadRequest, "session ID must be 1–80 letters, digits, '_' or '-'")
		return false
	}
	return true
}

func (s *chatAPI) chat(w http.ResponseWriter, r *http.Request) {
	if !validSession(w, r) {
		return
	}
	stream := true
	if value := r.URL.Query().Get("stream"); value != "" {
		if value != "true" && value != "false" {
			writeError(w, http.StatusBadRequest, "stream must be true or false")
			return
		}
		stream = value == "true"
	}
	var input *struct {
		Message string `json:"message"`
	}
	if !readJSON(w, r, &input) {
		return
	}
	if input == nil || strings.TrimSpace(input.Message) == "" || len(input.Message) > maxMessageBytes {
		writeError(w, http.StatusBadRequest, "message must contain 1–2048 bytes of non-blank text")
		return
	}
	session := r.PathValue("session")
	request := turnRequest{ID: string(sample.ID("chat-request")), Message: input.Message, Mode: s.mode, ExpiresAt: time.Now().Add(35 * time.Second)}
	var chunks <-chan streamChunk
	if stream {
		var unsubscribe func()
		var err error
		chunks, unsubscribe, err = s.relay.subscribe(request.ID)
		if err != nil {
			writeError(w, http.StatusTooManyRequests, "too many active streams")
			return
		}
		defer unsubscribe()
	}
	w.Header().Set("X-Chat-Request-ID", request.ID)
	w.Header().Set("Content-Location", "/chat/"+session+"/requests/"+request.ID)
	request, err := s.submit(r.Context(), session, "message", request)
	if err != nil {
		admissionError(w, err)
		return
	}
	if stream {
		if s.streamReply(w, r, session, request.ID, chunks) {
			s.acknowledge(session, request)
		}
		return
	}
	result, err := s.waitReceipt(r.Context(), session, request.ID)
	if err != nil {
		backendError(w, err)
		return
	}
	if result.Code != http.StatusOK {
		s.deliverJSON(w, session, request, result.Code, map[string]string{"error": result.Error})
		return
	}
	s.deliverJSON(w, session, request, http.StatusOK, chatResponse{SessionID: session, Message: result.Reply, Mode: s.mode})
}

func (s *chatAPI) history(w http.ResponseWriter, r *http.Request) {
	if !validSession(w, r) {
		return
	}
	state, err := s.store.State(r.Context(), r.PathValue("session"))
	if err != nil {
		backendError(w, err)
		return
	}
	if state == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	writeJSON(w, http.StatusOK, historyResponse{r.PathValue("session"), state.Messages, state.Mode})
}

func (s *chatAPI) reset(w http.ResponseWriter, r *http.Request) {
	if !validSession(w, r) {
		return
	}
	request := turnRequest{ID: string(sample.ID("chat-reset")), Mode: s.mode, ExpiresAt: time.Now().Add(35 * time.Second)}
	session := r.PathValue("session")
	w.Header().Set("X-Chat-Request-ID", request.ID)
	w.Header().Set("Content-Location", "/chat/"+session+"/requests/"+request.ID)
	request, err := s.submit(r.Context(), session, "reset", request)
	if err != nil {
		admissionError(w, err)
		return
	}
	result, err := s.waitReceipt(r.Context(), r.PathValue("session"), request.ID)
	if err != nil {
		backendError(w, err)
		return
	}
	if result.Code != http.StatusOK {
		s.deliverJSON(w, session, request, result.Code, map[string]string{"error": result.Error})
		return
	}
	s.deliverJSON(w, session, request, http.StatusOK, resetResponse{session, "reset"})
}

func (s *chatAPI) requestStatus(w http.ResponseWriter, r *http.Request) {
	if !validSession(w, r) {
		return
	}
	if !sessionPattern.MatchString(r.PathValue("request")) {
		writeError(w, http.StatusBadRequest, "invalid request ID")
		return
	}
	state, err := s.store.State(r.Context(), r.PathValue("session"))
	if err != nil {
		backendError(w, err)
		return
	}
	if state != nil {
		if result, ok := state.findReceipt(r.PathValue("request")); ok {
			writeJSON(w, http.StatusOK, result)
			return
		}
	}
	writeError(w, http.StatusNotFound, "receipt not found (queued, unknown, or outside the retention window)")
}

func (s *chatAPI) waitReceipt(ctx context.Context, session, id string) (receipt, error) {
	var result receipt
	err := sample.Until(ctx, 200*time.Millisecond, func() (bool, error) {
		state, err := s.store.State(ctx, session)
		if err != nil || state == nil {
			return false, err
		}
		var ok bool
		result, ok = state.findReceipt(id)
		return ok, nil
	})
	return result, err
}

func (s *chatAPI) streamReply(w http.ResponseWriter, r *http.Request, session, id string, chunks <-chan streamChunk) (delivered bool) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprint(w, ": waiting for durable entity\n\n"); err != nil {
		return
	}
	if err := http.NewResponseController(w).Flush(); err != nil {
		return
	}
	poll := time.NewTicker(200 * time.Millisecond)
	defer poll.Stop()
	heartbeat := time.NewTicker(5 * time.Second)
	defer heartbeat.Stop()
	var streamed strings.Builder
	for {
		select {
		case <-r.Context().Done():
			_ = sendSSE(w, sseEvent{Type: "error", Content: "HTTP wait canceled or timed out; the durable turn may still complete. Read history or the receipt URL."})
			return
		case chunk := <-chunks:
			if chunk.Offset != streamed.Len() {
				continue
			}
			if streamed.Len()+len(chunk.Content) > maxReplyBytes {
				_ = sendSSE(w, sseEvent{Type: "error", Content: "stream byte budget exceeded"})
				return
			}
			if err := sendSSE(w, sseEvent{Type: "chunk", Content: chunk.Content}); err != nil {
				return
			}
			streamed.WriteString(chunk.Content)
		case <-heartbeat.C:
			if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil && !errors.Is(err, http.ErrNotSupported) {
				return
			}
			if _, err := fmt.Fprint(w, ": working\n\n"); err != nil {
				return
			}
			if err := http.NewResponseController(w).Flush(); err != nil {
				return
			}
		case <-poll.C:
			state, err := s.store.State(r.Context(), session)
			if err != nil {
				_ = sendSSE(w, sseEvent{Type: "error", Content: "failed to read durable receipt"})
				return
			}
			if state == nil {
				continue
			}
			result, exists := state.findReceipt(id)
			if !exists {
				continue
			}
			if result.Code != http.StatusOK {
				return sendSSE(w, sseEvent{Type: "error", Content: result.Error}) == nil
			}
			if !strings.HasPrefix(result.Reply, streamed.String()) {
				_ = sendSSE(w, sseEvent{Type: "error", Content: "provisional response changed during a retry; read the committed reply from history"})
				return
			}
			for _, chunk := range splitChunks(result.Reply[streamed.Len():]) {
				if err := sendSSE(w, sseEvent{Type: "chunk", Content: chunk}); err != nil {
					return
				}
			}
			// Done is emitted only after DTS confirms the entity-state commit.
			return sendSSE(w, sseEvent{Type: "done"}) == nil
		}
	}
}

func sendSSE(w http.ResponseWriter, event sseEvent) error {
	controller := http.NewResponseController(w)
	if err := controller.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return err
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
		return err
	}
	return controller.Flush()
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
	_ = writeJSONResponse(w, code, value)
}

func writeJSONResponse(w http.ResponseWriter, code int, value any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, map[string]string{"error": message})
}

func backendError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		writeError(w, http.StatusGatewayTimeout, "HTTP wait timed out; durable work may still complete. Read history or the receipt URL.")
	case errors.Is(err, context.Canceled):
		writeError(w, http.StatusRequestTimeout, "HTTP wait canceled; durable work may still complete")
	default:
		writeError(w, http.StatusBadGateway, "DTS request failed")
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
		WriteTimeout: 45 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 * 1024,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}
	result := make(chan error, 1)
	go func() { result <- server.Serve(listener) }()
	fmt.Printf("Chat API listening on http://%s (until -timeout or Ctrl+C)\n", listener.Addr())
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
