package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/microsoft/durabletask-go/api"
	"github.com/microsoft/durabletask-go/exporthistory"
)

type fakeSource struct {
	ids   []api.InstanceID
	query api.InstanceIDQuery
	reads int
}

func (s *fakeSource) ListInstanceIDs(_ context.Context, query api.InstanceIDQuery) (*api.InstanceIDQueryResult, error) {
	s.query = query
	return &api.InstanceIDQueryResult{InstanceIDs: s.ids}, nil
}
func (s *fakeSource) FetchOrchestrationMetadata(_ context.Context, _ api.InstanceID, _ ...api.FetchOrchestrationMetadataOptions) (*api.OrchestrationMetadata, error) {
	s.reads++
	return &api.OrchestrationMetadata{}, nil
}
func (s *fakeSource) StreamOrchestrationHistory(_ context.Context, _ api.InstanceID, _ api.HistoryQuery, _ api.HistoryEventHandler) error {
	s.reads++
	return nil
}

func TestOwnershipGuardNeverReadsUnrelatedHistory(t *testing.T) {
	inner := &fakeSource{ids: []api.InstanceID{"mine"}}
	source := &ownedHistorySource{inner: inner, allowed: map[api.InstanceID]struct{}{"mine": {}}}
	ctx := context.Background()
	query := api.InstanceIDQuery{
		CompletedTimeFrom: time.Now().Add(-time.Minute), CompletedTimeTo: time.Now(),
		RuntimeStatus: []api.OrchestrationStatus{api.RUNTIME_STATUS_COMPLETED}, PageSize: 2,
	}
	if _, err := source.ListInstanceIDs(ctx, query); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(inner.query, query) {
		t.Fatal("the bounded query was not preserved")
	}
	inner.ids = append(inner.ids, "unrelated")
	if _, err := source.ListInstanceIDs(ctx, query); !errors.Is(err, errUnownedHistory) {
		t.Fatalf("mixed page was not rejected: %v", err)
	}
	if _, err := source.FetchOrchestrationMetadata(ctx, "unrelated"); !errors.Is(err, errUnownedHistory) {
		t.Fatalf("unrelated metadata read was not rejected: %v", err)
	}
	if err := source.StreamOrchestrationHistory(ctx, "unrelated", api.HistoryQuery{}, nil); !errors.Is(err, errUnownedHistory) {
		t.Fatalf("unrelated history read was not rejected: %v", err)
	}
	if inner.reads != 0 {
		t.Fatal("guard allowed an unrelated read to reach the SDK")
	}
}

type fakeStore struct{ writes int }

func (s *fakeStore) Write(_ context.Context, object exporthistory.ExportObject) error {
	s.writes++
	_, err := io.Copy(io.Discard, object.Content)
	return err
}

func TestStorageOwnershipGuard(t *testing.T) {
	inner := &fakeStore{}
	store := &ownedHistoryStore{
		inner: inner, container: "mine", prefix: "job/",
		allowed: map[api.InstanceID]struct{}{"mine": {}},
	}
	object := exporthistory.ExportObject{
		Container: "mine", Name: "job/history.jsonl.gz",
		Metadata: map[string]string{"instanceId": "mine"}, Content: bytes.NewReader(nil),
	}
	if err := store.Write(context.Background(), object); err != nil {
		t.Fatal(err)
	}
	object.Name = "other-job/history.jsonl.gz"
	if err := store.Write(context.Background(), object); err == nil {
		t.Fatal("wrong prefix accepted")
	}
	object.Name = "job/history.jsonl.gz"
	object.Metadata["instanceId"] = "other"
	if err := store.Write(context.Background(), object); err == nil {
		t.Fatal("unowned history accepted")
	}
	if inner.writes != 1 {
		t.Fatal("invalid writes reached the underlying store")
	}
}

func validHistory() ([]api.HistoryEvent, sourceExecution) {
	source := sourceExecution{ID: "go-source", ExecutionID: "execution", Input: 3}
	return []api.HistoryEvent{
		{Type: api.HistoryEventExecutionStarted, ExecutionStarted: &api.HistoryExecutionStartedEvent{
			InstanceID: source.ID, ExecutionID: source.ExecutionID, Name: orchestratorName, SerializedInput: "3",
		}},
		{Type: api.HistoryEventTaskScheduled, EventID: 7, TaskScheduled: &api.HistoryTaskScheduledEvent{
			Name: squareName, SerializedInput: "3",
		}},
		{Type: api.HistoryEventTaskCompleted, TaskCompleted: &api.HistoryTaskResultEvent{
			TaskScheduledID: 7, SerializedResult: "9",
		}},
		{Type: api.HistoryEventExecutionCompleted, ExecutionCompleted: &api.HistoryExecutionCompletedEvent{
			RuntimeStatus: api.RUNTIME_STATUS_COMPLETED, SerializedResult: "9",
		}},
	}, source
}

func gzipJSONL(t *testing.T, events []api.HistoryEvent) []byte {
	t.Helper()
	var body bytes.Buffer
	writer := gzip.NewWriter(&body)
	for _, event := range events {
		if err := json.NewEncoder(writer).Encode(event); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return body.Bytes()
}

func TestGzipJSONLContainsActualTargetExecution(t *testing.T) {
	events, source := validHistory()
	body := gzipJSONL(t, events)
	decoded, err := decodeJSONL(body)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyHistory(decoded, source); err != nil {
		t.Fatal(err)
	}
	source.ID = "another-instance"
	if err := verifyHistory(decoded, source); err == nil {
		t.Fatal("history from another instance accepted")
	}
	for _, bad := range [][]byte{body[:len(body)-4], []byte("not gzip"), gzipJSONL(t, nil)} {
		if _, err := decodeJSONL(bad); err == nil {
			t.Fatal("invalid or empty gzip stream accepted")
		}
	}
}

func TestRejectIncompleteOrCorruptHistory(t *testing.T) {
	for _, mutate := range []func([]api.HistoryEvent) []api.HistoryEvent{
		func(events []api.HistoryEvent) []api.HistoryEvent { return events[:3] },
		func(events []api.HistoryEvent) []api.HistoryEvent {
			events[2].TaskCompleted.SerializedResult = "8"
			return events
		},
		func(events []api.HistoryEvent) []api.HistoryEvent {
			events[2].TaskCompleted.TaskScheduledID = 99
			return events
		},
		func(events []api.HistoryEvent) []api.HistoryEvent {
			events[0].ExecutionStarted.ExecutionID = "wrong"
			return events
		},
	} {
		events, source := validHistory()
		if err := verifyHistory(mutate(events), source); err == nil {
			t.Fatal("invalid history accepted")
		}
	}
}

func TestMetadataAndIsolation(t *testing.T) {
	value := base64.RawURLEncoding.EncodeToString([]byte("go-source-你好"))
	got, err := decodeMetadataID(map[string]*string{"Instanceidbase64": &value}, "instanceIdBase64")
	if err != nil || got != "go-source-你好" {
		t.Fatalf("metadata decode: %q %v", got, err)
	}
	if _, err := decodeMetadataID(nil, "instanceIdBase64"); err == nil {
		t.Fatal("missing metadata accepted")
	}
	for _, flag := range []string{"", "true", "0"} {
		if requireIsolatedTaskHub(flag) == nil {
			t.Fatal("isolation must be explicitly acknowledged with 1")
		}
	}
	if err := requireIsolatedTaskHub("1"); err != nil {
		t.Fatal(err)
	}
}

func TestAzuriteStorage(t *testing.T) {
	t.Setenv("AZURE_STORAGE_CONNECTION_STRING", "")
	t.Setenv("AZURE_STORAGE_BLOB_ENDPOINT", "")
	options, err := storageOptions("go-export-test")
	if err != nil {
		t.Fatal(err)
	}
	if !options.AllowInsecureHTTP || options.ConnectionString != developmentStorage {
		t.Fatal("default does not use loopback Azurite")
	}
	if _, err := exporthistory.NewAzureBlobHistoryStore(options); err != nil {
		t.Fatal(err)
	}
}

func TestCompletionWindowCoversSourceLifetimes(t *testing.T) {
	start := time.Date(2026, 9, 15, 17, 59, 34, 0, time.UTC)
	sources := []sourceInstance{
		{CreatedAt: start.Add(989595 * time.Microsecond), CompletedAt: start.Add(time.Second + 419721500*time.Nanosecond)},
		{CreatedAt: start.Add(4*time.Second + 27390400*time.Nanosecond), CompletedAt: start.Add(4*time.Second + 455984700*time.Nanosecond)},
	}
	from, to := completionWindow(sources)
	if !from.Equal(start) || !to.Equal(start.Add(5*time.Second)) {
		t.Fatalf("window = [%s, %s), want [%s, %s)", from, to, start, start.Add(5*time.Second))
	}
	// Allow an index timestamp within the lifetime but outside tight metadata bounds.
	indexedCompletion := sources[0].CompletedAt.Add(-2 * time.Millisecond)
	if indexedCompletion.Before(from) || !indexedCompletion.Before(to) {
		t.Fatal("window excludes a completion within the source's lifetime")
	}
	reversedFrom, reversedTo := completionWindow([]sourceInstance{sources[1], sources[0]})
	if !reversedFrom.Equal(from) || !reversedTo.Equal(to) {
		t.Fatal("window depends on source order")
	}
}
