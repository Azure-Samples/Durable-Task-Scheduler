package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/microsoft/durabletask-go/api"
	"github.com/microsoft/durabletask-go/payload"
	"github.com/microsoft/durabletask-go/task"
)

const (
	record           = "RECORD|"
	smallRecords     = 10
	largeRecords     = 300_000
	thresholdBytes   = 64 * 1024
	grpcMessageBytes = 128 * 1024
	maxPayloadBytes  = 4 * 1024 * 1024
)

type payloadInput struct {
	Records int    `json:"records"`
	Content string `json:"content"`
	SHA256  string `json:"sha256"`
}

type payloadResult struct {
	Content string `json:"content"`
	Records int    `json:"records"`
	Bytes   int    `json:"bytes"`
	SHA256  string `json:"sha256"`
}

type payloadWorkflow struct {
	orchestrator string
	generate     string
	process      string
}

func newPayloadWorkflow(runID string) payloadWorkflow {
	return payloadWorkflow{
		orchestrator: "GoLargePayloadOrchestrator-" + runID,
		generate:     "GoLargePayloadGenerateData-" + runID,
		process:      "GoLargePayloadProcessData-" + runID,
	}
}

func (w payloadWorkflow) register(registry *task.TaskRegistry) error {
	if err := registry.AddOrchestratorN(w.orchestrator, w.orchestrate); err != nil {
		return err
	}
	if err := registry.AddActivityN(w.generate, generateData); err != nil {
		return err
	}
	return registry.AddActivityN(w.process, processData)
}

func main() {
	sample.Main("large-payload", run)
}

func run(ctx context.Context) (err error) {
	container := string(sample.ID("large-payload"))
	workflow := newPayloadWorkflow(container)
	storeOptions, blobClient, err := storageOptions(container)
	if err != nil {
		return err
	}
	store, err := payload.NewAzureBlobStore(storeOptions)
	if err != nil {
		return fmt.Errorf("configure payload store: %w", err)
	}
	options, err := sample.Options()
	if err != nil {
		return err
	}
	// A 2.1 MB payload cannot pass these deliberately smaller gRPC bounds inline.
	// The same options configure both the client and worker.
	options.MaxSendMessageSize = grpcMessageBytes
	options.MaxReceiveMessageSize = grpcMessageBytes
	options.LargePayloads = &api.LargePayloadOptions{
		Store:           store,
		Resolver:        store,
		ThresholdBytes:  thresholdBytes,
		MaxPayloadBytes: maxPayloadBytes,
	}
	registry := task.NewTaskRegistry()
	if err := workflow.register(registry); err != nil {
		return err
	}
	host, err := sample.Start(ctx, registry, options)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, host.Close()) }()
	fmt.Printf("Payload container: %s (retained for history hydration)\n", container)

	for _, count := range []int{smallRecords, largeRecords} {
		content, err := recordData(count)
		if err != nil {
			return err
		}
		input := payloadInput{Records: count, Content: content, SHA256: checksum([]byte(content))}
		id := sample.ID("large-payload")
		fmt.Printf("Instance: %s (%d records)\n", id, count)
		if _, err := host.Client.ScheduleNewOrchestration(ctx, workflow.orchestrator,
			api.WithInstanceID(id), api.WithInput(input)); err != nil {
			return err
		}
		var output payloadResult
		if err := sample.Wait(ctx, host.Client, id, &output); err != nil {
			return err
		}
		if err := verifyRoundTrip(input, output); err != nil {
			return err
		}
		names, err := listPayloadBlobs(ctx, blobClient, container)
		if err != nil {
			return err
		}
		if count == smallRecords {
			if len(names) != 0 {
				return fmt.Errorf("small payload must stay inline; found %d blobs", len(names))
			}
		} else {
			if len(names) < 4 {
				return fmt.Errorf("expected externalized input, activity data, and output; found only %d blobs", len(names))
			}
			for _, name := range names {
				if err := verifyPayloadBlob(ctx, blobClient, container, name, content); err != nil {
					return err
				}
			}
		}
		fmt.Printf("Verified %d records / %d bytes; SHA-256=%s; stored blobs=%d\n",
			count, output.Bytes, output.SHA256, len(names))
	}
	return nil
}

func (w payloadWorkflow) orchestrate(ctx *task.OrchestrationContext) (any, error) {
	var input payloadInput
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	var generated string
	if err := ctx.CallActivity(w.generate, task.WithActivityInput(input.Records)).Await(&generated); err != nil {
		return nil, err
	}
	if generated != input.Content || checksum([]byte(generated)) != input.SHA256 {
		return nil, errors.New("generated data differs from hydrated orchestration input")
	}
	var result payloadResult
	if err := ctx.CallActivity(w.process, task.WithActivityInput(generated)).Await(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func generateData(ctx task.ActivityContext) (any, error) {
	var count int
	if err := ctx.GetInput(&count); err != nil {
		return nil, err
	}
	return recordData(count)
}

func processData(ctx task.ActivityContext) (any, error) {
	var content string
	if err := ctx.GetInput(&content); err != nil {
		return nil, err
	}
	count := strings.Count(content, record)
	expected, err := recordData(count)
	if err != nil {
		return nil, err
	}
	if expected != content {
		return nil, errors.New("activity received corrupted record data")
	}
	return payloadResult{
		Content: content,
		Records: count,
		Bytes:   len(content),
		SHA256:  checksum([]byte(content)),
	}, nil
}

func recordData(count int) (string, error) {
	// Reserve space for JSON and the result's checksum before the SDK size cap.
	if count < 0 || count > (maxPayloadBytes-1024)/len(record) {
		return "", errors.New("record count exceeds the sample's payload limit")
	}
	return strings.Repeat(record, count), nil
}

func checksum(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func verifyRoundTrip(input payloadInput, output payloadResult) error {
	if output.Content != input.Content || output.Records != input.Records ||
		output.Bytes != len(input.Content) || output.SHA256 != input.SHA256 ||
		checksum([]byte(output.Content)) != input.SHA256 {
		return errors.New("client round-trip content, record count, size, or SHA-256 mismatch")
	}
	return nil
}

func listPayloadBlobs(ctx context.Context, client *azblob.Client, container string) ([]string, error) {
	var names []string
	pager := client.NewListBlobsFlatPager(container, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if bloberror.HasCode(err, bloberror.ContainerNotFound) {
			return names, nil // The SDK creates the container only on the first externalization.
		}
		if err != nil {
			return nil, fmt.Errorf("list payload blobs: %w", err)
		}
		for _, item := range page.Segment.BlobItems {
			if item.Name == nil {
				return nil, errors.New("blob listing omitted a name")
			}
			names = append(names, *item.Name)
		}
	}
	return names, nil
}

func verifyPayloadBlob(ctx context.Context, client *azblob.Client, container, name, wantContent string) error {
	properties, err := client.ServiceClient().NewContainerClient(container).NewBlobClient(name).GetProperties(ctx, nil)
	if err != nil {
		return fmt.Errorf("read payload blob properties: %w", err)
	}
	if properties.ContentEncoding == nil || !strings.EqualFold(*properties.ContentEncoding, "gzip") {
		return fmt.Errorf("payload blob %s is not stored with gzip encoding", name)
	}
	response, err := client.DownloadStream(ctx, container, name, nil)
	if err != nil {
		return fmt.Errorf("download payload blob: %w", err)
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, maxPayloadBytes+1))
	if err := errors.Join(readErr, response.Body.Close()); err != nil {
		return err
	}
	// Go's HTTP transport can already have decompressed Content-Encoding: gzip.
	if response.ContentEncoding != nil && strings.EqualFold(*response.ContentEncoding, "gzip") {
		reader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return err
		}
		body, readErr = io.ReadAll(io.LimitReader(reader, maxPayloadBytes+1))
		if err := errors.Join(readErr, reader.Close()); err != nil {
			return err
		}
	}
	return verifyStoredPayload(body, properties.Metadata, wantContent)
}

func verifyStoredPayload(body []byte, metadata map[string]*string, wantContent string) error {
	if len(body) <= thresholdBytes || len(body) > maxPayloadBytes {
		return fmt.Errorf("stored payload has invalid uncompressed size %d", len(body))
	}
	if metadataValue(metadata, "durabletask_size") != strconv.Itoa(len(body)) ||
		metadataValue(metadata, "durabletask_sha256") != checksum(body) {
		return errors.New("stored blob size or SHA-256 integrity metadata mismatch")
	}
	var content string
	if err := json.Unmarshal(body, &content); err != nil {
		var object struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(body, &object); err != nil {
			return fmt.Errorf("decode stored payload JSON: %w", err)
		}
		content = object.Content
	}
	if content != wantContent {
		return errors.New("downloaded blob does not contain the exact generated record bytes")
	}
	return nil
}

func metadataValue(metadata map[string]*string, key string) string {
	for name, value := range metadata {
		if strings.EqualFold(name, key) && value != nil {
			return *value
		}
	}
	return ""
}
