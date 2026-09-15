package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/microsoft/durabletask-go/api"
	"github.com/microsoft/durabletask-go/exporthistory"
)

func verifyExportBlobs(ctx context.Context, client *azblob.Client, container, prefix string, sources []sourceExecution) (int, error) {
	expected := make(map[api.InstanceID]sourceExecution, len(sources))
	for _, source := range sources {
		expected[source.ID] = source
	}
	seen := make(map[api.InstanceID]struct{}, len(sources))
	totalEvents := 0
	pager := client.NewListBlobsFlatPager(container, &azblob.ListBlobsFlatOptions{Prefix: &prefix})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return 0, err
		}
		for _, item := range page.Segment.BlobItems {
			if item.Name == nil {
				return 0, errors.New("export blob is missing a name")
			}
			name := *item.Name
			properties, err := client.ServiceClient().NewContainerClient(container).NewBlobClient(name).GetProperties(ctx, nil)
			if err != nil {
				return 0, err
			}
			instanceID, err := decodeMetadataID(properties.Metadata, "instanceIdBase64")
			if err != nil {
				return 0, err
			}
			source, owned := expected[api.InstanceID(instanceID)]
			if !owned {
				return 0, errUnownedHistory
			}
			if _, duplicate := seen[source.ID]; duplicate {
				return 0, errors.New("duplicate exported instance")
			}
			executionID, err := decodeMetadataID(properties.Metadata, "executionIdBase64")
			if err != nil {
				return 0, err
			}
			if executionID != source.ExecutionID ||
				metadataValue(properties.Metadata, "schemaVersion") != exporthistory.DefaultSchemaVersion {
				return 0, errors.New("export metadata execution ID or schema version mismatch")
			}
			digest := sha256.Sum256([]byte(source.CompletedAt.UTC().Format(time.RFC3339Nano) + "|" + string(source.ID)))
			if name != prefix+hex.EncodeToString(digest[:])+".jsonl.gz" {
				return 0, errors.New("export did not use the expected deterministic JSONL blob name")
			}
			if properties.ContentType == nil || *properties.ContentType != "application/gzip" ||
				(properties.ContentEncoding != nil && *properties.ContentEncoding != "") {
				return 0, errors.New("JSONL export must be an opaque application/gzip blob without Content-Encoding")
			}
			response, err := client.DownloadStream(ctx, container, name, nil)
			if err != nil {
				return 0, err
			}
			body, readErr := io.ReadAll(io.LimitReader(response.Body, maxHistoryBytes+1))
			if err := errors.Join(readErr, response.Body.Close()); err != nil {
				return 0, err
			}
			if len(body) > maxHistoryBytes {
				return 0, errors.New("compressed history exceeds the verification size bound")
			}
			events, err := decodeJSONL(body)
			if err != nil {
				return 0, fmt.Errorf("decode exported JSONL: %w", err)
			}
			if err := verifyHistory(events, source); err != nil {
				return 0, err
			}
			seen[source.ID] = struct{}{}
			totalEvents += len(events)
		}
	}
	if len(seen) != len(expected) {
		return 0, fmt.Errorf("downloaded %d owned histories, expected %d", len(seen), len(expected))
	}
	return totalEvents, nil
}

func decodeJSONL(body []byte) ([]api.HistoryEvent, error) {
	reader, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	plain, readErr := io.ReadAll(io.LimitReader(reader, maxHistoryBytes+1))
	if err := errors.Join(readErr, reader.Close()); err != nil {
		return nil, err
	}
	if len(plain) > maxHistoryBytes {
		return nil, errors.New("uncompressed history exceeds the verification size bound")
	}
	scanner := bufio.NewScanner(bytes.NewReader(plain))
	scanner.Buffer(make([]byte, 4096), maxHistoryBytes)
	var events []api.HistoryEvent
	for scanner.Scan() {
		var event api.HistoryEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, err
		}
		if len(events) == maxHistoryEvents {
			return nil, errors.New("history exceeds the verification event bound")
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, errors.New("empty exported history")
	}
	return events, nil
}

func verifyHistory(events []api.HistoryEvent, source sourceExecution) error {
	var starts, schedules, activities, completions int
	var scheduledID int32
	for _, event := range events {
		switch event.Type {
		case api.HistoryEventExecutionStarted:
			starts++
			started := event.ExecutionStarted
			if started == nil || started.InstanceID != source.ID || started.ExecutionID != source.ExecutionID ||
				started.Name != orchestratorName || !serializedIntEquals(started.SerializedInput, source.Input) {
				return errors.New("exported ExecutionStarted identity/input mismatch")
			}
		case api.HistoryEventTaskScheduled:
			schedules++
			if event.TaskScheduled == nil || event.TaskScheduled.Name != squareName ||
				!serializedIntEquals(event.TaskScheduled.SerializedInput, source.Input) {
				return errors.New("exported activity schedule/input mismatch")
			}
			scheduledID = event.EventID
		case api.HistoryEventTaskCompleted:
			activities++
			if schedules != 1 || event.TaskCompleted == nil || event.TaskCompleted.TaskScheduledID != scheduledID ||
				!serializedIntEquals(event.TaskCompleted.SerializedResult, source.Input*source.Input) {
				return errors.New("exported activity result/correlation mismatch")
			}
		case api.HistoryEventExecutionCompleted:
			completions++
			if event.ExecutionCompleted == nil || event.ExecutionCompleted.RuntimeStatus != api.RUNTIME_STATUS_COMPLETED ||
				!serializedIntEquals(event.ExecutionCompleted.SerializedResult, source.Input*source.Input) {
				return errors.New("exported terminal status/output mismatch")
			}
		case api.HistoryEventTaskFailed, api.HistoryEventExecutionTerminated:
			return errors.New("exported source history contains a failure")
		}
	}
	if starts != 1 || schedules != 1 || activities != 1 || completions != 1 {
		return fmt.Errorf("history lacks the expected start/activity/completion sequence: %d/%d/%d/%d",
			starts, schedules, activities, completions)
	}
	return nil
}

func serializedIntEquals(value string, want int) bool {
	var got int
	return json.Unmarshal([]byte(value), &got) == nil && got == want
}

func metadataValue(metadata map[string]*string, key string) string {
	for name, value := range metadata {
		if strings.EqualFold(name, key) && value != nil {
			return *value
		}
	}
	return ""
}

func decodeMetadataID(metadata map[string]*string, key string) (string, error) {
	encoded := metadataValue(metadata, key)
	if encoded == "" {
		return "", fmt.Errorf("export blob is missing %s metadata", key)
	}
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(encoded)
	if err != nil || len(decoded) == 0 {
		return "", fmt.Errorf("export blob has invalid %s metadata", key)
	}
	return string(decoded), nil
}
