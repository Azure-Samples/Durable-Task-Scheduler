package main

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/testutil"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

func TestIntegration(t *testing.T) {
	ctx := testutil.IntegrationContext(t)
	// Sharing a hub must not dispatch one run's work to another run's Blob store.
	results := make(chan error, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Go(func() { results <- exercisePayloadRoundTrips(ctx) })
	}
	workers.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Error(err)
		}
	}
}

func exercisePayloadRoundTrips(ctx context.Context) (err error) {
	worker, err := startWorker(ctx)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, worker.Close()) }()
	blobs, err := payloadBlobReader(worker.container)
	if err != nil {
		return err
	}
	for _, count := range []int{smallRecords, largeRecords} {
		content, err := recordData(count)
		if err != nil {
			return err
		}
		_, output, err := roundTrip(ctx, worker, content)
		if err != nil {
			return err
		}
		want := payloadExpectation{Records: count, Content: content, SHA256: checksum([]byte(content))}
		if err := verifyRoundTrip(want, output); err != nil {
			return err
		}
		names, err := listPayloadBlobs(ctx, blobs, worker.container)
		if err != nil {
			return err
		}
		if count == smallRecords {
			if err := testutil.Require(len(names) == 0, "small payload externalized to %d blobs", len(names)); err != nil {
				return err
			}
			continue
		}
		if err := testutil.Require(len(content) > grpcMessageBytes && len(content) >= 2*1024*1024,
			"large payload is only %d bytes", len(content)); err != nil {
			return err
		}
		if err := testutil.Require(len(names) >= 4, "expected input/activity/output blobs, found %d", len(names)); err != nil {
			return err
		}
		for _, name := range names {
			if err := verifyPayloadBlob(ctx, blobs, worker.container, name, content); err != nil {
				return err
			}
		}
	}
	return nil
}

func payloadBlobReader(container string) (*azblob.Client, error) {
	options, err := storageOptions(container)
	if err != nil {
		return nil, err
	}
	if options.ConnectionString != "" {
		return azblob.NewClientFromConnectionString(options.ConnectionString, nil)
	}
	return azblob.NewClient(options.AccountURL, options.Credential, nil)
}
