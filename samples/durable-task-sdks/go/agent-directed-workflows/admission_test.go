package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func reserveTestTurn(t *testing.T, state *chatState, operation string, input turnRequest) turnRequest {
	t.Helper()
	input.Operation = operation
	for index, slot := range state.Slots {
		if slot.RequestID != "" && time.Now().Before(slot.Until) {
			continue
		}
		input.Slot, input.Epoch = index, slot.Epoch
		ok, err := state.reserve(input, time.Now())
		if err != nil || !ok {
			t.Fatalf("reserve test turn: %v %v", ok, err)
		}
		input.Epoch++
		return input
	}
	t.Fatal("test needs an available receipt slot")
	return input
}

func TestProtectedReceiptsSurviveRecoveryCachePressure(t *testing.T) {
	a := &agent{mode: "real", model: modelFunc(func(_ context.Context, _ []modelMessage, emit func(string)) (modelMessage, error) {
		text := strings.Repeat("x", maxReplyBytes-16)
		emit(text)
		return modelMessage{Role: "assistant", Content: text}, nil
	})}
	state := chatState{}
	tickets := make([]turnRequest, receiptSlotCount)
	for index := range tickets {
		tickets[index] = reserveTestTurn(t, &state, "message", turnRequest{
			ID: fmt.Sprintf("active-%d", index), Mode: "real", Message: "test", ExpiresAt: time.Now().Add(time.Minute),
		})
		var result receipt
		state, result = a.applyTurn(context.Background(), state, "message", tickets[index])
		if result.Code != 200 {
			t.Fatal(result.Error)
		}
	}
	for index := range 100 {
		state.remember(receipt{ID: fmt.Sprintf("delivered-%d", index), Reply: strings.Repeat("y", maxReplyBytes), Code: 200})
	}
	for _, input := range tickets {
		result, ok := state.findReceipt(input.ID)
		if !ok || result.Code != 200 || len(result.Reply) != maxReplyBytes-16 {
			t.Fatalf("cache eviction lost an active committed receipt: %+v", result)
		}
	}
	if len(state.Receipts) > 16 || receiptBytes(state.Receipts) > 32*1024 {
		t.Fatal("recovery cache is unbounded")
	}
	for _, slot := range state.Slots {
		if slot.Result == nil || receiptBytes([]receipt{*slot.Result}) > maxProtectedReceiptBytes {
			t.Fatal("protected slot exceeded its byte budget")
		}
	}
}

func TestReservationFencesStaleOperationsAndExpiresBoundedly(t *testing.T) {
	state := chatState{}
	now := time.Now()
	a := &agent{mode: "mock"}
	first := reserveTestTurn(t, &state, "message", turnRequest{
		ID: "first", Mode: "mock", Message: "first", ExpiresAt: now.Add(time.Minute),
	})
	// A competing request based on the old snapshot cannot claim this slot.
	stale := first
	stale.ID, stale.Epoch = "loser", first.Epoch-1
	if accepted, err := state.reserve(stale, now); err != nil || accepted {
		t.Fatalf("stale compare-and-set succeeded: %v %v", accepted, err)
	}
	state, rejected := a.applyTurn(context.Background(), state, "message", stale)
	if rejected.Code != 429 || len(state.Messages) != 0 {
		t.Fatal("unadmitted request executed")
	}
	if state.acknowledge(first) {
		t.Fatal("pending work was acknowledged before a committed result")
	}
	state, completed := a.applyTurn(context.Background(), state, "message", first)
	if completed.Code != 200 || !state.acknowledge(first) {
		t.Fatal("completed request could not release its slot")
	}
	second := reserveTestTurn(t, &state, "message", turnRequest{
		ID: "second", Mode: "mock", Message: "second", ExpiresAt: now.Add(time.Minute),
	})
	if state.acknowledge(first) || !state.ownsSlot(second) {
		t.Fatal("delayed acknowledgement released a newer request")
	}
	// Simulate a disconnected HTTP caller: no acknowledgement, bounded lease.
	slot := state.Slots[second.Slot]
	replacement := second
	replacement.ID, replacement.Epoch = "after-expiry", slot.Epoch
	if accepted, err := state.reserve(replacement, slot.Until.Add(-time.Nanosecond)); err != nil || accepted {
		t.Fatal("unacknowledged receipt lease was reused before expiry")
	}
	if accepted, err := state.reserve(replacement, slot.Until); err != nil || !accepted {
		t.Fatalf("expired reservation was never reclaimable: %v %v", accepted, err)
	}
	if state.ownsSlot(second) {
		t.Fatal("expired execution token was not fenced")
	}
}

func TestSerializedProtectedReceiptBudget(t *testing.T) {
	a := &agent{mode: "real", model: modelFunc(func(_ context.Context, _ []modelMessage, emit func(string)) (modelMessage, error) {
		text := strings.Repeat("<", maxReplyBytes)
		emit(text)
		return modelMessage{Role: "assistant", Content: text}, nil
	})}
	state := chatState{}
	input := reserveTestTurn(t, &state, "message", turnRequest{
		ID: "escape-pressure", Mode: "real", Message: "test", ExpiresAt: time.Now().Add(time.Minute),
	})
	state, result := a.applyTurn(context.Background(), state, "message", input)
	if result.Code != 502 || len(state.Messages) != 0 || receiptBytes([]receipt{result}) > maxProtectedReceiptBytes {
		t.Fatalf("JSON escaping bypassed the protected byte budget: %+v", result)
	}
}

type pendingSignal struct {
	operation string
	input     turnRequest
}

// This fault-injection adapter publishes its first N operations as one batch.
// It tests application admission against delayed visibility, not DTS durability
// or an invented in-memory Go SDK backend. The parent runs backend stress.
type batchVisibilityStore struct {
	mu         sync.Mutex
	agent      *agent
	state      chatState
	pending    []pendingSignal
	firstBatch int
	committed  bool
	executed   map[string]bool
}

func (s *batchVisibilityStore) Signal(ctx context.Context, _ string, operation string, input turnRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.committed {
		s.pending = append(s.pending, pendingSignal{operation, input})
		if len(s.pending) < s.firstBatch {
			return nil
		}
		for _, signal := range s.pending {
			if err := s.apply(ctx, signal.operation, signal.input); err != nil {
				return err
			}
		}
		s.pending = nil
		s.committed = true
		return nil
	}
	return s.apply(ctx, operation, input)
}

func (s *batchVisibilityStore) apply(ctx context.Context, operation string, input turnRequest) error {
	switch operation {
	case "reserve":
		_, err := s.state.reserve(input, time.Now())
		return err
	case "ack":
		s.state.acknowledge(input)
		return nil
	case "message", "reset":
		var result receipt
		s.state, result = s.agent.applyTurn(ctx, s.state, operation, input)
		if result.Code == 200 {
			s.executed[input.ID] = true
		}
		return nil
	default:
		return errors.New("unexpected test operation")
	}
}

func (s *batchVisibilityStore) State(context.Context, string) (*chatState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.Marshal(s.state)
	if err != nil {
		return nil, err
	}
	var snapshot chatState
	err = json.Unmarshal(data, &snapshot)
	return &snapshot, err
}

func TestConcurrentHTTPReceiptsSurviveBatchedVisibility(t *testing.T) {
	for _, test := range []struct {
		name  string
		count int
		large bool
	}{
		{"seventeen-short-turns", 17, false},
		{"four-near-eight-KiB-replies", 4, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var modelCalls atomic.Int32
			replyFor := func(text string) string {
				if test.large {
					return strings.Repeat("x", maxReplyBytes-16)
				}
				return "Echo: " + text
			}
			// Deliberately no local relay publication: exercise cross-process
			// committed-receipt fallback for both SSE and JSON callers.
			a := &agent{mode: "real", model: modelFunc(func(_ context.Context, messages []modelMessage, emit func(string)) (modelMessage, error) {
				modelCalls.Add(1)
				text := replyFor(messages[len(messages)-1].Content)
				emit(text)
				return modelMessage{Role: "assistant", Content: text}, nil
			})}
			store := &batchVisibilityStore{
				agent: a, firstBatch: test.count, state: chatState{Mode: "real", Messages: []message{}, Receipts: []receipt{}},
				executed: map[string]bool{},
			}
			app := &chatAPI{store: store, mode: "real", relay: newRelay()}
			server := httptest.NewServer(app.handler())
			defer server.Close()
			type outcome struct {
				id   string
				code int
				err  error
			}
			results := make(chan outcome, test.count)
			start := make(chan struct{})
			for index := range test.count {
				go func() {
					<-start
					text := fmt.Sprintf("turn-%d", index)
					body, _ := json.Marshal(map[string]string{"message": text})
					address := server.URL + "/chat/stress"
					if index%2 == 0 {
						address += "?stream=false"
					}
					client := &http.Client{Timeout: 10 * time.Second}
					response, err := client.Post(address, "application/json", strings.NewReader(string(body)))
					if err != nil {
						results <- outcome{err: err}
						return
					}
					defer response.Body.Close()
					got := outcome{id: response.Header.Get("X-Chat-Request-ID"), code: response.StatusCode}
					switch response.StatusCode {
					case http.StatusTooManyRequests:
						if response.Header.Get("Retry-After") != "1" {
							got.err = errors.New("backpressure response has no retry advice")
						}
					case http.StatusOK:
						var reply string
						if index%2 == 0 {
							var result chatResponse
							got.err = json.NewDecoder(response.Body).Decode(&result)
							reply = result.Message
						} else {
							reply, _, got.err = readSSE(response.Body)
						}
						if got.err == nil && reply != replyFor(text) {
							got.err = errors.New("committed reply was lost or truncated")
						}
					default:
						got.err = fmt.Errorf("unexpected HTTP status %d", response.StatusCode)
					}
					results <- got
				}()
			}
			close(start)
			succeeded := 0
			for range test.count {
				got := <-results
				if got.err != nil {
					t.Error(got.err)
					continue
				}
				store.mu.Lock()
				executed := store.executed[got.id]
				store.mu.Unlock()
				if got.code == 200 {
					succeeded++
					if !executed {
						t.Error("HTTP success preceded a committed turn")
					}
				} else if executed {
					t.Error("HTTP 429 request executed a turn")
				}
			}
			if succeeded == 0 || int(modelCalls.Load()) != succeeded {
				t.Fatalf("a completed/model-executed turn lost its caller: model=%d delivered=%d", modelCalls.Load(), succeeded)
			}
			state, err := store.State(context.Background(), "stress")
			if err != nil || len(state.Messages) != 2*succeeded || receiptBytes(state.Receipts) > 32*1024 {
				t.Fatalf("history/cache invariants failed: %+v %v", state, err)
			}
		})
	}
}

func TestCapacityBackpressurePrecedesExecution(t *testing.T) {
	app := testAPI()
	store := app.store.(*memoryTestStore)
	state := chatState{}
	for index := range receiptSlotCount {
		reserveTestTurn(t, &state, "message", turnRequest{
			ID: fmt.Sprintf("unacknowledged-%d", index), Mode: "mock", Message: "pending", ExpiresAt: time.Now().Add(time.Minute),
		})
	}
	store.states["busy"] = state
	request := httptest.NewRequest(http.MethodPost, "/chat/busy", strings.NewReader(`{"message":"must not execute"}`))
	request.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.handler().ServeHTTP(w, request)
	if w.Code != 429 || w.Header().Get("Retry-After") != "1" || strings.Contains(w.Body.String(), `"type":"chunk"`) {
		t.Fatalf("admission pressure was not an explicit pre-stream HTTP 429: %d %s", w.Code, w.Body.String())
	}
	after, err := store.State(context.Background(), "busy")
	if err != nil || len(after.Messages) != 0 {
		t.Fatal("backpressured request changed conversation history")
	}
}

func TestRecoveryReadCannotReleaseAnotherActiveCaller(t *testing.T) {
	app := testAPI()
	store := app.store.(*memoryTestStore)
	state := chatState{}
	input := reserveTestTurn(t, &state, "message", turnRequest{
		ID: "active-owner", Mode: "mock", Message: "keep me", ExpiresAt: time.Now().Add(time.Minute),
	})
	state, result := store.agent.applyTurn(context.Background(), state, "message", input)
	if result.Code != 200 {
		t.Fatal(result.Error)
	}
	store.states["session"] = state
	w := httptest.NewRecorder()
	app.handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/chat/session/requests/active-owner", nil))
	after, err := store.State(context.Background(), "session")
	if w.Code != 200 || err != nil || !after.ownsSlot(input) || len(after.Receipts) != 0 {
		t.Fatal("a recovery reader acknowledged a receipt still needed by its original HTTP owner")
	}
}

type failedDeliveryWriter struct{ header http.Header }

func (w *failedDeliveryWriter) Header() http.Header       { return w.header }
func (w *failedDeliveryWriter) WriteHeader(int)           {}
func (w *failedDeliveryWriter) Write([]byte) (int, error) { return 0, errors.New("delivery failed") }

func TestFailedHTTPDeliveryRetainsProtectedReceipt(t *testing.T) {
	app := testAPI()
	store := app.store.(*memoryTestStore)
	state := chatState{}
	input := reserveTestTurn(t, &state, "message", turnRequest{
		ID: "disconnected", Mode: "mock", Message: "recover me", ExpiresAt: time.Now().Add(time.Minute),
	})
	state, result := store.agent.applyTurn(context.Background(), state, "message", input)
	store.states["session"] = state
	app.deliverJSON(&failedDeliveryWriter{header: make(http.Header)}, "session", input, 200, result)
	after, err := store.State(context.Background(), "session")
	if err != nil || !after.ownsSlot(input) {
		t.Fatal("failed delivery released an undelivered receipt")
	}
}

type delayedAdmissionStore struct {
	pending    turnRequest
	operations []string
}

func (s *delayedAdmissionStore) Signal(_ context.Context, _ string, operation string, input turnRequest) error {
	s.operations = append(s.operations, operation)
	s.pending = input
	return nil
}

func (s *delayedAdmissionStore) State(context.Context, string) (*chatState, error) {
	if s.pending.ID != "" {
		return nil, context.DeadlineExceeded
	}
	return &chatState{}, nil
}

func TestTimedOutAdmissionCannotExecuteAfterHTTPBackpressure(t *testing.T) {
	store := &delayedAdmissionStore{}
	app := &chatAPI{store: store, mode: "mock"}
	_, err := app.submit(context.Background(), "session", "message", turnRequest{
		ID: "late-reservation", Mode: "mock", Message: "never execute", ExpiresAt: time.Now().Add(time.Minute),
	})
	if !errors.Is(err, errBackpressure) || len(store.operations) != 1 || store.operations[0] != "reserve" {
		t.Fatalf("ambiguous admission caused execution: %v %v", store.operations, err)
	}
	state := chatState{}
	accepted, err := state.reserve(store.pending, time.Now())
	if err != nil || !accepted || len(state.Messages) != 0 || state.Slots[store.pending.Slot].Result != nil {
		t.Fatal("a late reservation executed work without a separate execution signal")
	}
}
