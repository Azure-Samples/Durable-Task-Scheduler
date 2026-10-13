package main

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/microsoft/durabletask-go/api"
)

type fakeHistoryReader struct {
	history    *api.OrchestrationHistory
	err        error
	instanceID api.InstanceID
	query      api.HistoryQuery
	calls      int
}

func (c *fakeHistoryReader) GetOrchestrationHistory(_ context.Context, id api.InstanceID, query api.HistoryQuery) (*api.OrchestrationHistory, error) {
	c.instanceID, c.query = id, query
	c.calls++
	return c.history, c.err
}

func checkpointHistory(t *testing.T, metadata *api.OrchestrationMetadata, checkpoint researchState) *api.OrchestrationHistory {
	t.Helper()
	input, err := json.Marshal(checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	return &api.OrchestrationHistory{
		InstanceID: metadata.InstanceID, ExecutionID: metadata.ExecutionID,
		Events: []*api.HistoryEvent{
			{Type: api.HistoryEventOrchestratorStarted},
			{Type: api.HistoryEventExecutionStarted, ExecutionStarted: &api.HistoryExecutionStartedEvent{
				Name: researchName, InstanceID: metadata.InstanceID, ExecutionID: metadata.ExecutionID,
				SerializedInput: string(input),
			}},
		},
	}
}

func TestFixtureCheckpointUsesCurrentExecutionHistory(t *testing.T) {
	metadata := metadataFor(t, api.RUNTIME_STATUS_COMPLETED)
	var original researchState
	if err := metadata.ReadInput(&original); err != nil {
		t.Fatal(err)
	}
	if original.Iteration != 0 || len(original.Findings) != 0 || !reflect.DeepEqual(original.Queries, []string{demoTopic}) {
		t.Fatalf("regression fixture must retain the initial metadata input: %+v", original)
	}
	_, continued := fixtureResult(t)
	reader := &fakeHistoryReader{history: checkpointHistory(t, metadata, continued)}
	if err := verifyFixtureCheckpoint(context.Background(), reader, metadata); err != nil {
		t.Fatalf("continued checkpoint rejected because metadata retains initial input: %v", err)
	}
	if reader.calls != 1 || reader.instanceID != metadata.InstanceID ||
		reader.query != (api.HistoryQuery{ExecutionID: "execution-2", MaxEvents: 200, MaxBytes: 1024 * 1024}) {
		t.Fatalf("history read was not bounded and execution-ID-pinned: %+v", reader)
	}
}

func TestFixtureCheckpointCannotFallBackToMetadataInput(t *testing.T) {
	metadata := metadataFor(t, api.RUNTIME_STATUS_COMPLETED)
	var original researchState
	if err := metadata.ReadInput(&original); err != nil {
		t.Fatal(err)
	}
	_, continued := fixtureResult(t)
	input, err := json.Marshal(continued)
	if err != nil {
		t.Fatal(err)
	}
	metadata.SerializedInput = string(input)
	reader := &fakeHistoryReader{history: checkpointHistory(t, metadata, original)}
	if err := verifyFixtureCheckpoint(context.Background(), reader, metadata); err == nil {
		t.Fatal("advanced metadata input masked an uncontinued execution history")
	}
}

func TestFixtureCheckpointRejectsMissingOrMixedExecutionEvidence(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*api.OrchestrationMetadata, *api.OrchestrationHistory)
	}{
		{"missing selected execution", func(m *api.OrchestrationMetadata, h *api.OrchestrationHistory) { m.ExecutionID = "" }},
		{"wrong metadata task", func(m *api.OrchestrationMetadata, h *api.OrchestrationHistory) { m.Name = paperName }},
		{"wrong history instance", func(m *api.OrchestrationMetadata, h *api.OrchestrationHistory) { h.InstanceID = "other" }},
		{"stale history execution", func(m *api.OrchestrationMetadata, h *api.OrchestrationHistory) { h.ExecutionID = "execution-1" }},
		{"missing history execution", func(m *api.OrchestrationMetadata, h *api.OrchestrationHistory) { h.ExecutionID = "" }},
		{"wrong start instance", func(m *api.OrchestrationMetadata, h *api.OrchestrationHistory) {
			h.Events[1].ExecutionStarted.InstanceID = "other"
		}},
		{"stale start execution", func(m *api.OrchestrationMetadata, h *api.OrchestrationHistory) {
			h.Events[1].ExecutionStarted.ExecutionID = "execution-1"
		}},
		{"wrong start name", func(m *api.OrchestrationMetadata, h *api.OrchestrationHistory) {
			h.Events[1].ExecutionStarted.Name = paperName
		}},
		{"missing start detail", func(m *api.OrchestrationMetadata, h *api.OrchestrationHistory) { h.Events[1].ExecutionStarted = nil }},
		{"missing input", func(m *api.OrchestrationMetadata, h *api.OrchestrationHistory) {
			h.Events[1].ExecutionStarted.SerializedInput = ""
		}},
		{"malformed input", func(m *api.OrchestrationMetadata, h *api.OrchestrationHistory) {
			h.Events[1].ExecutionStarted.SerializedInput = "{"
		}},
		{"null input", func(m *api.OrchestrationMetadata, h *api.OrchestrationHistory) {
			h.Events[1].ExecutionStarted.SerializedInput = "null"
		}},
		{"no start", func(m *api.OrchestrationMetadata, h *api.OrchestrationHistory) { h.Events = h.Events[:1] }},
		{"duplicate start", func(m *api.OrchestrationMetadata, h *api.OrchestrationHistory) {
			h.Events = append(h.Events, h.Events[1])
		}},
		{"nil event", func(m *api.OrchestrationMetadata, h *api.OrchestrationHistory) { h.Events = append(h.Events, nil) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			metadata := metadataFor(t, api.RUNTIME_STATUS_COMPLETED)
			_, continued := fixtureResult(t)
			history := checkpointHistory(t, metadata, continued)
			test.change(metadata, history)
			reader := &fakeHistoryReader{history: history}
			if err := verifyFixtureCheckpoint(context.Background(), reader, metadata); err == nil {
				t.Fatal("invalid execution evidence passed checkpoint verification")
			}
			if metadata.ExecutionID == "" && reader.calls != 0 {
				t.Fatal("missing execution ID triggered an unpinned history read")
			}
		})
	}
	metadata := metadataFor(t, api.RUNTIME_STATUS_COMPLETED)
	reader := &fakeHistoryReader{}
	if err := verifyFixtureCheckpoint(context.Background(), reader, nil); err == nil || reader.calls != 0 {
		t.Fatal("missing metadata was accepted")
	}
	if err := verifyFixtureCheckpoint(context.Background(), reader, metadata); err == nil {
		t.Fatal("missing history was accepted")
	}
	reader.err = errors.New("history unavailable")
	if err := verifyFixtureCheckpoint(context.Background(), reader, metadata); !errors.Is(err, reader.err) {
		t.Fatalf("history failure was not propagated: %v", err)
	}
}

func TestFixtureCheckpointPreservesAllCarriedState(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*researchState)
	}{
		{"iteration", func(s *researchState) { s.Iteration = 0 }},
		{"topic", func(s *researchState) { s.Topic = "another topic" }},
		{"mode", func(s *researchState) { s.Mode = "real" }},
		{"iteration budget", func(s *researchState) { s.MaxIterations = 3 }},
		{"queries", func(s *researchState) { s.Queries[0] = "another query" }},
		{"findings", func(s *researchState) { s.Findings = nil }},
		{"analysis", func(s *researchState) { s.Findings[0].Summary = "missing prior analysis" }},
		{"paper IDs", func(s *researchState) { s.Findings[0].PaperIDs = nil }},
		{"papers", func(s *researchState) { s.Papers = nil }},
		{"paper metadata", func(s *researchState) { s.Papers[0].Title = "altered title" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			metadata := metadataFor(t, api.RUNTIME_STATUS_COMPLETED)
			_, continued := fixtureResult(t)
			test.change(&continued)
			reader := &fakeHistoryReader{history: checkpointHistory(t, metadata, continued)}
			if err := verifyFixtureCheckpoint(context.Background(), reader, metadata); err == nil {
				t.Fatal("lost or altered carried-forward state passed checkpoint verification")
			}
		})
	}
}

func TestResearchStatusUsesCurrentProgressWithOriginalMetadataInput(t *testing.T) {
	metadata := metadataFor(t, api.RUNTIME_STATUS_RUNNING)
	current := progress{Mode: "fixture", Phase: "researching", Iteration: 2,
		FindingsCount: 1, PaperIDs: []string{"fixture-001", "fixture-002"}}
	encoded, err := json.Marshal(current)
	if err != nil {
		t.Fatal(err)
	}
	metadata.SerializedCustomStatus = string(encoded)
	status, err := researchStatus(metadata)
	if err != nil || status.Topic != demoTopic || status.Mode != "fixture" || !reflect.DeepEqual(status.progress, current) {
		t.Fatalf("original metadata input overrode current progress: %+v %v", status, err)
	}
	metadata.RuntimeStatus = api.RUNTIME_STATUS_COMPLETED
	status, err = researchStatus(metadata)
	if err != nil || status.Iteration != 2 || status.FindingsCount != 3 || status.Report != expectedDemoReport {
		t.Fatalf("completed output did not override initial input/earlier progress: %+v %v", status, err)
	}
}
