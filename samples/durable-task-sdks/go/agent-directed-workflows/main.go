package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
	"github.com/microsoft/durabletask-go/task"
)

var (
	serve  = flag.Bool("serve", false, "Serve the interactive API instead of the bounded mock demonstration")
	listen = flag.String("listen", "127.0.0.1:5000", "Loopback listen address for -serve")
	mode   = flag.String("mode", defaultMode(), "Agent mode: mock (explicit echo) or real (Azure OpenAI; requires -serve)")
)

func defaultMode() string {
	if value := strings.TrimSpace(os.Getenv("CHAT_MODE")); value != "" {
		return value
	}
	return "mock"
}

func main() { sample.Main("agent-directed-workflows", run) }

func run(ctx context.Context) error {
	if *serve {
		if _, err := loopbackAddress(*listen); err != nil {
			return err
		}
	}
	if *mode != "mock" && *mode != "real" {
		return errors.New("-mode must be mock or real")
	}
	if !*serve && *mode != "mock" {
		return errors.New("the bounded verification demo uses mock mode; use -serve -mode real for Azure OpenAI")
	}
	if !*serve {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 65*time.Second)
		defer cancel()
	}
	relay := newRelay()
	agent := &agent{mode: *mode, relay: relay}
	if *mode == "real" {
		config, err := loadModelConfig(os.Getenv)
		if err != nil {
			return err
		}
		model, err := newOpenAIModel(config)
		if err != nil {
			return err
		}
		agent.model = model
	}
	fmt.Printf("Chat mode: %s (mock is an echo, and the weather tool always uses synthetic data)\n", *mode)
	registry := task.NewTaskRegistry()
	if err := registry.AddEntityN(entityName, agent.entity); err != nil {
		return err
	}
	return sample.WithHost(ctx, registry, func(ctx context.Context, client *dts.Client) error {
		app := &chatAPI{store: schedulerStore{client}, mode: *mode, relay: relay}
		if *serve {
			return serveHTTP(ctx, *listen, app.handler())
		}
		return demo(ctx, app)
	})
}

func demo(ctx context.Context, app *chatAPI) error {
	server := httptest.NewServer(app.handler())
	defer server.Close()
	client := &http.Client{Timeout: 40 * time.Second}
	session := string(sample.ID("chat-session"))
	base := server.URL + "/chat/" + session
	send := func(text string) error {
		var reply chatResponse
		code, headers, err := requestJSON(ctx, client, http.MethodPost, base+"?stream=false",
			map[string]string{"message": text}, &reply)
		if err != nil {
			return err
		}
		return sample.Require(code == 200 && reply.SessionID == session && reply.Message == "Echo: "+text &&
			reply.Mode == "mock" && headers.Get("X-Chat-Mode") == "mock" && headers.Get("X-Chat-Request-ID") != "",
			"invalid chat response: %d %+v", code, reply)
	}
	if err := send("Remember: Ada."); err != nil {
		return err
	}
	payload := `{"message":"What did I say?"}`
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, base, strings.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode != 200 || response.Header.Get("Content-Type") != "text/event-stream" ||
		response.Header.Get("Cache-Control") != "no-cache" || response.Header.Get("X-Chat-Mode") != "mock" {
		response.Body.Close()
		return errors.New("invalid streaming HTTP status or headers")
	}
	reply, chunks, err := readSSE(response.Body)
	response.Body.Close()
	if err != nil {
		return err
	}
	if err := sample.Require(reply == "Echo: What did I say?" && chunks >= 2, "unexpected SSE response %q (%d chunks)", reply, chunks); err != nil {
		return err
	}
	history := historyResponse{}
	code, _, err := requestJSON(ctx, client, http.MethodGet, base+"/history", nil, &history)
	if err != nil {
		return err
	}
	expected := []message{
		{"user", "Remember: Ada."}, {"assistant", "Echo: Remember: Ada."},
		{"user", "What did I say?"}, {"assistant", "Echo: What did I say?"},
	}
	if err := sample.Require(code == 200 && reflect.DeepEqual(history.History, expected), "history lost a turn: %+v", history); err != nil {
		return err
	}
	// Two HTTP requests race, but the durable entity must persist whole turns serially.
	errorsCh := make(chan error, 2)
	for _, text := range []string{"Concurrent A", "Concurrent B"} {
		go func() { errorsCh <- send(text) }()
	}
	for range 2 {
		if err := <-errorsCh; err != nil {
			return err
		}
	}
	code, _, err = requestJSON(ctx, client, http.MethodGet, base+"/history", nil, &history)
	if err != nil {
		return err
	}
	if err := sample.Require(code == 200 && len(history.History) == 8 &&
		reflect.DeepEqual(history.History[:4], expected), "concurrent requests lost history: %+v", history); err != nil {
		return err
	}
	seen := map[string]bool{}
	for index := 4; index < 8; index += 2 {
		user, assistant := history.History[index], history.History[index+1]
		if err := sample.Require(user.Role == "user" && assistant.Role == "assistant" &&
			(user.Content == "Concurrent A" || user.Content == "Concurrent B") && !seen[user.Content] &&
			assistant.Content == "Echo: "+user.Content, "turns were interleaved: %+v", history); err != nil {
			return err
		}
		seen[user.Content] = true
	}
	// Read the entity directly, not through an HTTP cache.
	persisted, err := app.store.State(ctx, session)
	if err != nil {
		return err
	}
	if err := sample.Require(persisted != nil && reflect.DeepEqual(persisted.Messages, history.History),
		"HTTP history differs from durable entity state"); err != nil {
		return err
	}
	var reset resetResponse
	code, _, err = requestJSON(ctx, client, http.MethodPost, base+"/reset", nil, &reset)
	if err != nil {
		return err
	}
	if err := sample.Require(code == 200 && reset.SessionID == session && reset.Status == "reset", "invalid reset acknowledgement"); err != nil {
		return err
	}
	code, _, err = requestJSON(ctx, client, http.MethodGet, base+"/history", nil, &history)
	if err != nil {
		return err
	}
	if err := sample.Require(code == 200 && history.History != nil && len(history.History) == 0, "reset did not clear durable history"); err != nil {
		return err
	}
	if err := send("New conversation."); err != nil {
		return err
	}
	code, _, err = requestJSON(ctx, client, http.MethodGet, base+"/history", nil, &history)
	if err != nil {
		return err
	}
	if err := sample.Require(code == 200 && reflect.DeepEqual(history.History, []message{
		{"user", "New conversation."}, {"assistant", "Echo: New conversation."},
	}), "new conversation retained old messages"); err != nil {
		return err
	}
	code, _, err = requestJSON(ctx, client, http.MethodGet,
		server.URL+"/chat/"+string(sample.ID("chat-missing"))+"/history", nil, nil)
	if err != nil {
		return err
	}
	if err := sample.Require(code == http.StatusNotFound, "missing session returned %d", code); err != nil {
		return err
	}
	return sample.PrintJSON(map[string]any{"mode": "mock", "sessionId": session, "verified_turns": 5, "reset_verified": true, "history": history.History})
}

func readSSE(reader io.Reader) (string, int, error) {
	scanner := bufio.NewScanner(io.LimitReader(reader, 128*1024))
	scanner.Buffer(make([]byte, 4096), 32*1024)
	var text strings.Builder
	chunks, done := 0, false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data: ") || done {
			return "", 0, errors.New("invalid SSE framing or events after done")
		}
		var event sseEvent
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event); err != nil {
			return "", 0, err
		}
		switch event.Type {
		case "chunk":
			text.WriteString(event.Content)
			chunks++
		case "done":
			done = true
		case "error":
			return "", 0, fmt.Errorf("agent stream failed: %s", event.Content)
		default:
			return "", 0, fmt.Errorf("unknown SSE event %q", event.Type)
		}
	}
	if err := scanner.Err(); err != nil {
		return "", 0, err
	}
	if !done {
		return "", 0, errors.New("SSE stream ended without a committed done event")
	}
	return text.String(), chunks, nil
}

func requestJSON(ctx context.Context, client *http.Client, method, address string, input, output any) (int, http.Header, error) {
	var body io.Reader
	if input != nil {
		data, err := json.Marshal(input)
		if err != nil {
			return 0, nil, err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, address, body)
	if err != nil {
		return 0, nil, err
	}
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 128*1024+1))
	if err != nil {
		return 0, nil, err
	}
	if len(data) > 128*1024 || response.Header.Get("Content-Type") != "application/json" {
		return 0, nil, errors.New("invalid JSON HTTP response")
	}
	if output != nil {
		if err := json.Unmarshal(data, output); err != nil {
			return response.StatusCode, response.Header, err
		}
	}
	return response.StatusCode, response.Header, nil
}
