package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

const (
	receiptSlotCount         = 2
	maxProtectedReceiptBytes = 16 * 1024
	receiptLease             = 2 * time.Minute
	admissionTimeout         = 5 * time.Second
	admissionPollInterval    = 50 * time.Millisecond
)

var errBackpressure = errors.New("chat session is busy; no turn executed, retry later")

type receiptSlot struct {
	Epoch     uint64    `json:"epoch"`
	RequestID string    `json:"request_id,omitempty"`
	Operation string    `json:"operation,omitempty"`
	Until     time.Time `json:"protected_until"`
	Result    *receipt  `json:"result,omitempty"`
}

func (s chatState) ownsSlot(input turnRequest) bool {
	if input.Slot < 0 || input.Slot >= len(s.Slots) || input.ID == "" {
		return false
	}
	slot := s.Slots[input.Slot]
	return slot.RequestID == input.ID && slot.Epoch == input.Epoch
}

func (s *chatState) reserve(input turnRequest, now time.Time) (bool, error) {
	if input.ID == "" || len(input.ID) > 80 || input.ExpiresAt.IsZero() ||
		(input.Operation != "message" && input.Operation != "reset") ||
		input.Slot < 0 || input.Slot >= len(s.Slots) {
		return false, errors.New("invalid chat reservation")
	}
	slot := &s.Slots[input.Slot]
	if slot.Epoch != input.Epoch || (slot.RequestID != "" && now.Before(slot.Until)) {
		return false, nil
	}
	if slot.Epoch == ^uint64(0) {
		return false, errors.New("receipt slot generation exhausted")
	}
	if slot.Result != nil {
		s.remember(*slot.Result)
	}
	*slot = receiptSlot{
		Epoch: slot.Epoch + 1, RequestID: input.ID, Operation: input.Operation, Until: now.Add(receiptLease),
	}
	return true, nil
}

func (s *chatState) acknowledge(input turnRequest) bool {
	if !s.ownsSlot(input) {
		return false
	}
	slot := &s.Slots[input.Slot]
	if slot.Result == nil {
		return false
	}
	s.remember(*slot.Result)
	// Retaining the generation fences delayed reserve/execute/ack signals.
	*slot = receiptSlot{Epoch: slot.Epoch}
	return true
}

// Admission has no model side effects. Execution is signaled separately, only
// after a committed grant is observed. Even an ambiguous admission timeout can
// therefore return pre-execution backpressure safely.
func (s *chatAPI) submit(ctx context.Context, session, operation string, input turnRequest) (turnRequest, error) {
	input.Operation = operation
	admitCtx, cancel := context.WithTimeout(ctx, admissionTimeout)
	defer cancel()
	reserved, err := s.acquire(admitCtx, session, input)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
			return input, errBackpressure
		}
		return input, err
	}
	if err := s.store.Signal(ctx, session, operation, reserved); err != nil {
		// Execution may already have been accepted. Keep its protected slot and
		// recovery URL; do not turn this ambiguous failure into HTTP 429.
		return reserved, err
	}
	return reserved, nil
}

func (s *chatAPI) acquire(ctx context.Context, session string, input turnRequest) (turnRequest, error) {
	for {
		if err := ctx.Err(); err != nil {
			return input, err
		}
		state, err := s.store.State(ctx, session)
		if err != nil {
			return input, err
		}
		if state == nil {
			state = &chatState{}
		}
		now := time.Now()
		candidate := -1
		for index, slot := range state.Slots {
			if slot.RequestID == "" || !now.Before(slot.Until) {
				candidate = index
				break
			}
		}
		if candidate < 0 {
			if err := waitAdmission(ctx); err != nil {
				return input, err
			}
			continue
		}
		input.Slot, input.Epoch = candidate, state.Slots[candidate].Epoch
		if err := s.store.Signal(ctx, session, "reserve", input); err != nil {
			return input, err
		}
		for {
			current, err := s.store.State(ctx, session)
			if err != nil {
				return input, err
			}
			if current != nil {
				slot := current.Slots[candidate]
				if slot.Epoch > input.Epoch {
					if slot.RequestID == input.ID && slot.Epoch == input.Epoch+1 {
						input.Epoch = slot.Epoch
						return input, nil
					}
					// A different request won this generation. The old reserve
					// can never execute, so retrying another slot is safe.
					break
				}
			}
			if err := waitAdmission(ctx); err != nil {
				return input, err
			}
		}
	}
}

func waitAdmission(ctx context.Context) error {
	timer := time.NewTimer(admissionPollInterval)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *chatAPI) acknowledge(session string, input turnRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := s.store.Signal(ctx, session, "ack", input); err != nil {
		log.Printf("Chat receipt acknowledgement outcome is unknown for %s; use its protected slot or bounded recovery cache", input.ID)
	}
}

func (s *chatAPI) deliverJSON(w http.ResponseWriter, session string, input turnRequest, code int, value any) {
	if err := writeJSONResponse(w, code, value); err != nil {
		log.Printf("Chat response delivery failed for %s; retaining its protected receipt", input.ID)
		return
	}
	if err := http.NewResponseController(w).Flush(); err != nil {
		log.Printf("Chat response flush failed for %s; retaining its protected receipt", input.ID)
		return
	}
	s.acknowledge(session, input)
}

func admissionError(w http.ResponseWriter, err error) {
	if errors.Is(err, errBackpressure) {
		w.Header().Set("Retry-After", "1")
		writeError(w, http.StatusTooManyRequests, err.Error())
		return
	}
	backendError(w, fmt.Errorf("chat admission/execution request: %w", err))
}
