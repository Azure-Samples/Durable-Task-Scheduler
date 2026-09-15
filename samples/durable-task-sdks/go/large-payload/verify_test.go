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

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
)

type payloadExpectation struct {
	Records int
	Content string
	SHA256  string
}

func checksum(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func verifyRoundTrip(input payloadExpectation, output payloadResult) error {
	if output.Content != input.Content || output.Records != input.Records ||
		output.Bytes != len(input.Content) ||
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
