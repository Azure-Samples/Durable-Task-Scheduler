package main

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/microsoft/durabletask-go/api"
	"github.com/microsoft/durabletask-go/payload"
	"github.com/microsoft/durabletask-go/task"
)

type activityInput struct{ value any }

func (a activityInput) Context() context.Context { return context.Background() }
func (a activityInput) GetInput(target any) error {
	body, err := json.Marshal(a.value)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

func TestConcurrentRunsHaveDisjointTaskRegistrations(t *testing.T) {
	registered := make(map[string]struct{})
	for _, runID := range []string{"go-large-payload-first", "go-large-payload-second"} {
		workflow := newPayloadWorkflow(runID)
		if workflow != newPayloadWorkflow(runID) {
			t.Fatal("task names must remain stable for one run, including orchestration replay")
		}
		registry := task.NewTaskRegistry()
		if err := workflow.register(registry); err != nil {
			t.Fatal(err)
		}
		snapshot := registry.Snapshot()
		if len(snapshot.Orchestrators) != 1 || len(snapshot.Activities) != 2 || len(snapshot.Entities) != 0 {
			t.Fatalf("unexpected registry: %+v", snapshot)
		}
		expected := map[string]bool{
			workflow.orchestrator: false,
			workflow.generate:     false,
			workflow.process:      false,
		}
		tasks := append(snapshot.Orchestrators, snapshot.Activities...)
		for _, registration := range tasks {
			if _, ok := expected[registration.Name]; !ok {
				t.Fatalf("registered a name not used by this run: %s", registration.Name)
			}
			expected[registration.Name] = true
			if !strings.HasPrefix(registration.Name, "GoLargePayload") ||
				!strings.HasSuffix(registration.Name, runID) || registration.Version != "" {
				t.Fatalf("registration is not scoped to this run: %+v", registration)
			}
			key := strings.ToLower(registration.Name)
			if _, shared := registered[key]; shared {
				t.Fatalf("two workers could accept the same work item: %s", registration.Name)
			}
			registered[key] = struct{}{}
		}
		for name, found := range expected {
			if !found {
				t.Fatalf("a scheduling/call name is not registered: %s", name)
			}
		}
	}
}

func TestGenerateProcessAndVerify(t *testing.T) {
	for _, count := range []int{0, smallRecords, largeRecords} {
		generated, err := generateData(activityInput{count})
		if err != nil {
			t.Fatal(err)
		}
		content := generated.(string)
		value, err := processData(activityInput{content})
		if err != nil {
			t.Fatal(err)
		}
		result := value.(payloadResult)
		input := payloadInput{Records: count, Content: content, SHA256: checksum([]byte(content))}
		if err := verifyRoundTrip(input, result); err != nil {
			t.Fatal(err)
		}
		result.Content += "!"
		if err := verifyRoundTrip(input, result); err == nil {
			t.Fatal("corruption was accepted")
		}
	}
	if largeRecords*len(record) <= grpcMessageBytes || largeRecords*len(record) < 2*1024*1024 {
		t.Fatal("large input no longer exceeds the sample's gRPC bounds by a substantial margin")
	}
}

func TestRejectInvalidRecordData(t *testing.T) {
	for _, count := range []int{-1, maxPayloadBytes} {
		if _, err := recordData(count); err == nil {
			t.Fatalf("invalid count %d accepted", count)
		}
	}
	for _, value := range []any{"RECORD|broken", 123} {
		if _, err := processData(activityInput{value}); err == nil {
			t.Fatalf("invalid payload %v accepted", value)
		}
	}
}

func TestVerifyActualStoredBytes(t *testing.T) {
	content, err := recordData(largeRecords)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []any{content, payloadInput{Content: content}, payloadResult{Content: content}} {
		body, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		size, digest := strconv.Itoa(len(body)), checksum(body)
		metadata := map[string]*string{"Durabletask_Size": &size, "Durabletask_Sha256": &digest}
		if err := verifyStoredPayload(body, metadata, content); err != nil {
			t.Fatal(err)
		}
		if err := verifyStoredPayload(body, metadata, content+"!"); err == nil {
			t.Fatal("wrong blob content accepted")
		}
		digest = "corrupt"
		if err := verifyStoredPayload(body, metadata, content); err == nil {
			t.Fatal("wrong blob checksum accepted")
		}
	}
	if err := verifyStoredPayload([]byte(`"tiny"`), nil, "tiny"); err == nil {
		t.Fatal("inline-sized payload accepted as externalized evidence")
	}
}

func TestStorageDefaultAndSharedConfiguration(t *testing.T) {
	t.Setenv("AZURE_STORAGE_CONNECTION_STRING", "")
	t.Setenv("AZURE_STORAGE_BLOB_ENDPOINT", "")
	options, client, err := storageOptions("go-large-payload-test")
	if err != nil {
		t.Fatal(err)
	}
	if options.ConnectionString != developmentStorage || !options.AllowInsecureHTTP ||
		client.URL() != "http://127.0.0.1:10000/devstoreaccount1/" {
		t.Fatalf("unexpected local Blob options: URL=%s allowHTTP=%t", client.URL(), options.AllowInsecureHTTP)
	}
	store, err := payload.NewAzureBlobStore(options)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := api.NormalizeLargePayloadOptions(&api.LargePayloadOptions{
		Store: store, Resolver: store, ThresholdBytes: thresholdBytes, MaxPayloadBytes: maxPayloadBytes,
	}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AZURE_STORAGE_CONNECTION_STRING", "UseDevelopmentStorage=true")
	if _, _, err := storageOptions("go-large-payload-test"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AZURE_STORAGE_BLOB_ENDPOINT", "https://example.blob.core.windows.net")
	if _, _, err := storageOptions("go-large-payload-test"); err == nil {
		t.Fatal("ambiguous storage authentication was accepted")
	}
}
