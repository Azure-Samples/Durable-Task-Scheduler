package main

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/microsoft/durabletask-go/api"
	"github.com/microsoft/durabletask-go/payload"
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

func TestEchoProcessAndVerify(t *testing.T) {
	for _, count := range []int{0, smallRecords, largeRecords} {
		content, err := recordData(count)
		if err != nil {
			t.Fatal(err)
		}
		echoed, err := echoData(activityInput{content})
		if err != nil {
			t.Fatal(err)
		}
		value, err := processData(activityInput{echoed})
		if err != nil {
			t.Fatal(err)
		}
		result := value.(payloadResult)
		input := payloadExpectation{Records: count, Content: content, SHA256: checksum([]byte(content))}
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

func TestPayloadWorkflowNames(t *testing.T) {
	workflow := newPayloadWorkflow()
	if workflow.orchestrator != "GoLargePayloadOrchestrator" ||
		workflow.echo != "GoLargePayloadEchoData" ||
		workflow.process != "GoLargePayloadProcessData" {
		t.Fatalf("unexpected task names: %+v", workflow)
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
	for _, value := range []any{content, payloadResult{Content: content}} {
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
	options, err := storageOptions("go-large-payload-test")
	if err != nil {
		t.Fatal(err)
	}
	client, err := payloadBlobReader("go-large-payload-test")
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
	if _, err := storageOptions("go-large-payload-test"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AZURE_STORAGE_BLOB_ENDPOINT", "https://example.blob.core.windows.net")
	if _, err := storageOptions("go-large-payload-test"); err == nil {
		t.Fatal("ambiguous storage authentication was accepted")
	}
}
