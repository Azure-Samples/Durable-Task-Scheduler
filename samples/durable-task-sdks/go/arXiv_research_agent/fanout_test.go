package main

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/microsoft/durabletask-go/task"
)

// These controlled Task futures exercise the SDK's real WhenAll combinator;
// they are not an in-memory orchestration backend.
type controlledTask struct {
	value    any
	err      error
	entered  chan struct{}
	complete <-chan struct{}
	drained  bool
	decoded  bool
}

func (f *controlledTask) Await(target any) error {
	if target == nil {
		if f.entered != nil {
			close(f.entered)
		}
		if f.complete != nil {
			<-f.complete
		}
		f.drained = true
		return f.err
	}
	if !f.drained {
		return errors.New("decoded a fan-out result before draining")
	}
	f.decoded = true
	data, err := json.Marshal(f.value)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func TestFanoutFailureWaitsForEverySibling(t *testing.T) {
	for _, kind := range []string{"query", "paper"} {
		t.Run(kind, func(t *testing.T) {
			failure := errors.New("first sibling failed")
			release := make(chan struct{})
			first := &controlledTask{err: failure}
			second := &controlledTask{entered: make(chan struct{}), complete: release}
			third := &controlledTask{err: errors.New("later sibling failed")}
			tasks := []task.Task{first, second, third}
			finished := make(chan error, 1)
			go func() {
				ctx := &task.OrchestrationContext{}
				if kind == "query" {
					_, err := collectFanoutResults[queryResult](ctx, tasks)
					finished <- err
				} else {
					_, err := collectFanoutResults[paper](ctx, tasks)
					finished <- err
				}
			}()
			select {
			case <-second.entered:
			case err := <-finished:
				close(release)
				t.Fatalf("failed before draining the second sibling: %v", err)
			case <-time.After(time.Second):
				close(release)
				t.Fatal("did not reach the pending sibling")
			}
			select {
			case err := <-finished:
				close(release)
				t.Fatalf("returned before the remaining sibling completed: %v", err)
			default:
			}
			close(release)
			if err := <-finished; !errors.Is(err, failure) {
				t.Fatalf("first failure was lost: %v", err)
			}
			for _, future := range []*controlledTask{first, second, third} {
				if !future.drained || future.decoded {
					t.Fatalf("failed batch was not fully drained before propagation: %+v", future)
				}
			}
		})
	}
}

func TestFanoutResultOrderAndDecodeFailure(t *testing.T) {
	ctx := &task.OrchestrationContext{}
	first := &controlledTask{value: paper{ID: "first"}}
	second := &controlledTask{value: paper{ID: "second"}}
	papers, err := collectFanoutResults[paper](ctx, []task.Task{first, second})
	if err != nil || !reflect.DeepEqual(paperIDs(papers), []string{"first", "second"}) {
		t.Fatalf("input-order aggregation changed: %v %v", papers, err)
	}
	invalid := &controlledTask{value: "not a paper"}
	sibling := &controlledTask{value: paper{ID: "finished"}}
	if _, err := collectFanoutResults[paper](ctx, []task.Task{invalid, sibling}); err == nil {
		t.Fatal("invalid result was accepted")
	}
	if !invalid.drained || !sibling.drained {
		t.Fatal("decode failure propagated before draining all siblings")
	}
}
