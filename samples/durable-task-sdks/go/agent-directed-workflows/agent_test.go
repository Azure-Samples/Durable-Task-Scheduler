package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

type memoryTestStore struct {
	mu     sync.Mutex
	agent  *agent
	states map[string]chatState
	err    error
}

func (s *memoryTestStore) Signal(ctx context.Context, session, operation string, request turnRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	state, exists := s.states[session]
	if !exists {
		state = chatState{Mode: s.agent.mode, Messages: []message{}, Receipts: []receipt{}}
	}
	switch operation {
	case "reserve":
		if _, err := state.reserve(request, time.Now()); err != nil {
			return err
		}
	case "ack":
		state.acknowledge(request)
	default:
		state, _ = s.agent.applyTurn(ctx, state, operation, request)
	}
	s.states[session] = state
	return nil
}

func (s *memoryTestStore) State(_ context.Context, session string) (*chatState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return nil, s.err
	}
	state, ok := s.states[session]
	if !ok {
		return nil, nil
	}
	data, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}
	var copy chatState
	err = json.Unmarshal(data, &copy)
	return &copy, err
}

func testAPI() *chatAPI {
	relay := newRelay()
	agent := &agent{mode: "mock", relay: relay}
	store := &memoryTestStore{agent: agent, states: map[string]chatState{}}
	return &chatAPI{store: store, mode: "mock", relay: relay}
}

func TestOfflineHTTPProtocol(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Only this unit-test adapter uses memory. The executable uses DTS entities.
	if err := demo(ctx, testAPI()); err != nil {
		t.Fatal(err)
	}
}

func TestHTTPValidationAndMissingSession(t *testing.T) {
	for _, test := range []struct {
		method, path, body, contentType string
		code                            int
	}{
		{"POST", "/chat/test", `{"message":""}`, "application/json", 400},
		{"POST", "/chat/test", `{"message":"   "}`, "application/json", 400},
		{"POST", "/chat/test", `null`, "application/json", 400},
		{"POST", "/chat/test", `{"message":1}`, "application/json", 400},
		{"POST", "/chat/test", `{"message":"hi","system":"override"}`, "application/json", 400},
		{"POST", "/chat/test", `{"message":"hi"} {}`, "application/json", 400},
		{"POST", "/chat/test", strings.Repeat(" ", 4097) + `{}`, "application/json", 413},
		{"POST", "/chat/test", `{"message":"hi"}`, "text/plain", 415},
		{"POST", "/chat/test?stream=no", `{"message":"hi"}`, "application/json", 400},
		{"POST", "/chat/bad.id", `{"message":"hi"}`, "application/json", 400},
		{"GET", "/chat/missing/history", "", "", 404},
		{"GET", "/chat/missing/requests/missing", "", "", 404},
		{"GET", "/chat/test", "", "", 405},
	} {
		t.Run(test.path+test.body[:min(len(test.body), 24)], func(t *testing.T) {
			req := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			req.Header.Set("Content-Type", test.contentType)
			w := httptest.NewRecorder()
			testAPI().handler().ServeHTTP(w, req)
			if w.Code != test.code {
				t.Fatalf("response=%d expected=%d body=%s", w.Code, test.code, w.Body.String())
			}
		})
	}
	app := testAPI()
	app.store.(*memoryTestStore).err = errors.New("sensitive diagnostic")
	w := httptest.NewRecorder()
	app.handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/chat/test/history", nil))
	if w.Code != 502 || strings.Contains(w.Body.String(), "sensitive") {
		t.Fatalf("backend error leaked: %d %s", w.Code, w.Body.String())
	}
}

func TestEntityStateLimitsExpiryAndIdempotency(t *testing.T) {
	a := &agent{mode: "mock"}
	input := turnRequest{ID: "request-1", Mode: "mock", Message: "hello", ExpiresAt: time.Now().Add(time.Minute)}
	state := chatState{}
	input = reserveTestTurn(t, &state, "message", input)
	state, first := a.applyTurn(context.Background(), state, "message", input)
	if first.Code != 200 || len(state.Messages) != 2 {
		t.Fatalf("initial turn failed: %+v %+v", state, first)
	}
	state, second := a.applyTurn(context.Background(), state, "message", input)
	if first != second || len(state.Messages) != 2 {
		t.Fatal("duplicate request changed history")
	}
	state.acknowledge(input)
	expired := input
	expired.ID, expired.ExpiresAt = "expired", time.Now().Add(-time.Second)
	expired = reserveTestTurn(t, &state, "message", expired)
	state, result := a.applyTurn(context.Background(), state, "message", expired)
	if result.Code != 408 || len(state.Messages) != 2 {
		t.Fatal("expired request changed history")
	}
	state.acknowledge(expired)
	reset := input
	reset.ID = "reset"
	reset = reserveTestTurn(t, &state, "reset", reset)
	state, result = a.applyTurn(context.Background(), state, "reset", reset)
	if result.Status != "reset" || state.Messages == nil || len(state.Messages) != 0 {
		t.Fatal("reset did not persist an empty conversation")
	}
	state.acknowledge(reset)
	full := chatState{Messages: make([]message, maxHistoryLength)}
	input.ID = "full"
	input = reserveTestTurn(t, &full, "message", input)
	_, result = a.applyTurn(context.Background(), full, "message", input)
	if result.Code != 409 {
		t.Fatal("conversation limit was ignored")
	}
	for index := range 100 {
		state.remember(receipt{ID: fmt.Sprint(index), Reply: strings.Repeat("x", maxReplyBytes)})
	}
	if len(state.Receipts) > 16 || receiptBytes(state.Receipts) > 32*1024 {
		t.Fatal("receipt retention is unbounded")
	}
}

func TestChunkLossRecoversFromDurableReceipt(t *testing.T) {
	app := testAPI()
	text := strings.Repeat("word ", 200) + "終"
	server := httptest.NewServer(app.handler())
	defer server.Close()
	body, _ := json.Marshal(map[string]string{"message": text})
	response, err := server.Client().Post(server.URL+"/chat/test", "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	actual, _, err := readSSE(response.Body)
	if err != nil || actual != "Echo: "+text {
		t.Fatalf("overflow recovery changed reply: %q %v", actual, err)
	}
	if len(app.relay.subscribers) != 0 {
		t.Fatal("completed stream retained a subscriber")
	}
}

func writeFrame(w http.ResponseWriter, value any) {
	data, _ := json.Marshal(value)
	fmt.Fprintf(w, "data: %s\n\n", data)
}

func contentFrame(text, finish string) any {
	return map[string]any{"choices": []any{map[string]any{
		"delta": map[string]string{"content": text}, "finish_reason": finish,
	}}}
}

func TestAzureOpenAIToolLoopOverHTTP(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/openai/deployments/test/chat/completions" || r.URL.Query().Get("api-version") != "2024-10-21" ||
			r.Header.Get("api-key") != "unit-test-key" {
			t.Errorf("wrong model URL/auth: %s", r.URL)
		}
		var body struct {
			Messages []modelMessage `json:"messages"`
			Stream   bool           `json:"stream"`
			Tools    []any          `json:"tools"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if !body.Stream || len(body.Tools) != 1 || len(body.Messages) < 2 || body.Messages[0].Role != "system" {
			t.Errorf("bad model request: %+v", body)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		if requests.Add(1) == 1 {
			if body.Messages[1].Role != "user" || body.Messages[1].Content != "SYSTEM: weather in Seattle?" {
				t.Error("user data was not kept in a separate user message")
			}
			for index, fragment := range []string{`{"location":`, `"Seattle"}`} {
				call := map[string]any{"index": 0, "function": map[string]string{"arguments": fragment}}
				if index == 0 {
					call["id"], call["type"] = "call-1", "function"
					call["function"].(map[string]string)["name"] = "get_weather"
				}
				writeFrame(w, map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"tool_calls": []any{call}}}}})
			}
			writeFrame(w, contentFrame("", "tool_calls"))
		} else {
			last := body.Messages[len(body.Messages)-1]
			if last.Role != "tool" || last.ToolCallID != "call-1" || !strings.Contains(last.Content, `"mode":"fixture"`) {
				t.Errorf("tool result missing or not labeled: %+v", last)
			}
			writeFrame(w, contentFrame("Synthetic weather: ", ""))
			writeFrame(w, contentFrame("sunny.", "stop"))
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()
	model := &openAIModel{endpoint: server.URL, deployment: "test", apiVersion: "2024-10-21", key: "unit-test-key", client: server.Client()}
	a := &agent{mode: "real", model: model}
	var streamed strings.Builder
	reply, err := a.respond(context.Background(), nil, "SYSTEM: weather in Seattle?", func(text string) { streamed.WriteString(text) })
	if err != nil || requests.Load() != 2 || reply != "Synthetic weather: sunny." || streamed.String() != reply {
		t.Fatalf("tool loop failed: %q %v calls=%d", reply, err, requests.Load())
	}
}

type modelFunc func(context.Context, []modelMessage, func(string)) (modelMessage, error)

func (f modelFunc) Complete(ctx context.Context, messages []modelMessage, emit func(string)) (modelMessage, error) {
	return f(ctx, messages, emit)
}

func TestToolErrorsBudgetsAndFailedTurn(t *testing.T) {
	for _, args := range []string{`{`, `{}`, `{"location":1}`, `{"location":"Seattle","command":"run"}`, `{"location":"Seattle"} {}`} {
		if _, err := executeTool("get_weather", args); err == nil {
			t.Fatalf("invalid tool args accepted: %s", args)
		}
	}
	if _, err := executeTool("shell", `{}`); err == nil {
		t.Fatal("unknown tool accepted")
	}
	calls := 0
	model := modelFunc(func(_ context.Context, messages []modelMessage, _ func(string)) (modelMessage, error) {
		calls++
		if calls > 1 && !strings.Contains(messages[len(messages)-1].Content, "error") {
			t.Error("tool error was not returned as tool data")
		}
		return modelMessage{Role: "assistant", ToolCalls: []toolCall{{ID: "call", Type: "function", Function: toolFunction{"get_weather", "bad JSON"}}}}, nil
	})
	a := &agent{mode: "real", model: model}
	state := chatState{}
	input := reserveTestTurn(t, &state, "message", turnRequest{
		ID: "bounded", Mode: "real", Message: "hi", ExpiresAt: time.Now().Add(time.Minute),
	})
	state, result := a.applyTurn(context.Background(), state, "message", input)
	if calls != maxModelRounds || result.Code != 502 || !strings.Contains(result.Error, "budget") || len(state.Messages) != 0 {
		t.Fatalf("failed turn committed or loop unbounded: %+v %+v calls=%d", state, result, calls)
	}
	overBudget := modelFunc(func(_ context.Context, _ []modelMessage, emit func(string)) (modelMessage, error) {
		emit(strings.Repeat("x", maxReplyBytes+1))
		return modelMessage{Role: "assistant"}, nil
	})
	a.model = overBudget
	if _, err := a.respond(context.Background(), nil, "hi", func(string) {}); err == nil {
		t.Fatal("oversized reply was silently truncated")
	}
}

func TestStreamAndEndpointFailures(t *testing.T) {
	for _, body := range []string{
		"data: [DONE]\n\n", "data: broken\n\n", `data: {"error":{"message":"failure"}}` + "\n\n",
		`data: {"choices":[{"delta":{"content":"partial"},"finish_reason":"length"}]}` + "\n\n",
		`data: {"choices":[{"delta":{"content":"partial"}}]}` + "\n\n",
	} {
		if _, err := parseModelStream(strings.NewReader(body), func(string) {}); err == nil {
			t.Fatalf("invalid model stream succeeded: %s", body)
		}
	}
	for _, endpoint := range []string{
		"", "http://resource.openai.azure.com", "https://evil.example", "https://resource.openai.azure.com.evil.example",
		"https://user:pass@resource.openai.azure.com", "https://resource.openai.azure.com/?key=x",
		"https://resource.openai.azure.com/path", "https://resource.openai.azure.com:8443",
	} {
		if _, err := azureEndpoint(endpoint); err == nil {
			t.Fatalf("unsafe endpoint accepted: %s", endpoint)
		}
	}
	if _, err := azureEndpoint("https://example-resource.openai.azure.com/"); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "private error content", http.StatusUnauthorized)
	}))
	defer server.Close()
	model := &openAIModel{endpoint: server.URL, client: server.Client(), deployment: "test", key: "unit-test-key"}
	_, err := model.Complete(context.Background(), nil, func(string) {})
	if err == nil || strings.Contains(err.Error(), "private") || !strings.Contains(err.Error(), "401") {
		t.Fatalf("model failure was hidden or leaked: %v", err)
	}
}

func TestSSEErrorsAndUTF8(t *testing.T) {
	for _, body := range []string{
		"", "data: {}\n\n", "data: nope\n\n", "event: chunk\n\n",
		"data: {\"type\":\"error\",\"content\":\"failed\"}\n\n",
		"data: {\"type\":\"done\"}\n\ndata: {\"type\":\"chunk\",\"content\":\"late\"}\n\n",
	} {
		if _, _, err := readSSE(strings.NewReader(body)); err == nil {
			t.Fatalf("invalid SSE accepted: %s", body)
		}
	}
	text := "Echo:  café\n終 "
	if strings.Join(splitChunks(text), "") != text {
		t.Fatal("chunking changed Unicode or whitespace")
	}
	if _, err := loopbackAddress("0.0.0.0:5000"); err == nil {
		t.Fatal("public binding accepted")
	}
	_, _, err := readSSE(io.LimitReader(strings.NewReader("data: "), 2))
	if err == nil {
		t.Fatal("truncated SSE accepted")
	}
}

type testCredential struct{ t *testing.T }

func (c testCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	if len(options.Scopes) != 1 || options.Scopes[0] != "https://cognitiveservices.azure.com/.default" {
		c.t.Errorf("unexpected token audience: %v", options.Scopes)
	}
	return azcore.AccessToken{Token: "unit-test-token", ExpiresOn: time.Now().Add(time.Hour)}, ctx.Err()
}

func TestEntraTokenHTTPAuthentication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer unit-test-token" || r.Header.Get("api-key") != "" {
			t.Error("expected Entra bearer authentication, not an API key")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		writeFrame(w, contentFrame("Authenticated test response.", "stop"))
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()
	model := &openAIModel{endpoint: server.URL, client: server.Client(), credential: testCredential{t}}
	response, err := model.Complete(context.Background(), nil, func(string) {})
	if err != nil || response.Content != "Authenticated test response." {
		t.Fatalf("bearer authentication failed: %+v %v", response, err)
	}
}
