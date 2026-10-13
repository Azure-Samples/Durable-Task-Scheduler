package main

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/testutil"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
)

type metadataClient interface {
	FetchOrchestrationMetadata(context.Context, api.InstanceID, ...api.FetchOrchestrationMetadataOptions) (*api.OrchestrationMetadata, error)
}

type queryClient interface {
	QueryInstances(context.Context, api.OrchestrationQuery) (*api.OrchestrationQueryResult, error)
}

type purgeClient interface {
	metadataClient
	PurgeInstances(context.Context, api.PurgeInstancesRequest) (*api.PurgeInstancesResult, error)
}

func TestIntegration(t *testing.T) {
	ctx := testutil.IntegrationContext(t)
	if err := verifyManagement(ctx); err != nil {
		t.Fatal(err)
	}
}

func verifyManagement(ctx context.Context) (err error) {
	host, err := startWorker(ctx)
	if err != nil {
		return err
	}
	var owned []api.InstanceID
	defer func() {
		if err != nil {
			err = errors.Join(err, stopOwned(host.Client, owned))
		}
		err = errors.Join(err, host.Close())
	}()
	c := host.Client
	prefix := string(sample.ID("management")) + "-"
	createdFrom := time.Now().UTC().Add(-time.Second)
	inputs := []batchInput{
		{BatchID: "batch-1", ItemCount: 10},
		{BatchID: "batch-2", ItemCount: 20},
		{BatchID: "batch-3", ItemCount: 30},
	}
	for i, input := range inputs {
		id := api.InstanceID(fmt.Sprintf("%sbatch-%d", prefix, i+1))
		owned = append(owned, id)
		if _, err := c.ScheduleNewOrchestration(ctx, workflowName,
			api.WithInstanceID(id), api.WithInput(input)); err != nil {
			return err
		}
	}
	for i, input := range inputs {
		if err := waitForBatch(ctx, c, owned[i], input); err != nil {
			return err
		}
	}

	original, err := c.FetchOrchestrationMetadata(ctx, owned[0], api.WithFetchPayloads(true))
	if err != nil {
		return err
	}
	restarted, err := c.RestartInstance(ctx, owned[0])
	if err != nil {
		return err
	}
	if restarted != owned[0] {
		return fmt.Errorf("same-ID restart returned %s, want %s", restarted, owned[0])
	}
	if err := waitForNewExecution(ctx, c, restarted, original.ExecutionID); err != nil {
		return err
	}
	if err := waitForBatch(ctx, c, restarted, inputs[0]); err != nil {
		return err
	}

	preserved, err := c.FetchOrchestrationMetadata(ctx, owned[1], api.WithFetchPayloads(true))
	if err != nil {
		return err
	}
	newID, err := c.RestartInstance(ctx, owned[1], api.WithRestartNewInstanceID(true))
	if err != nil {
		return err
	}
	if newID == api.EmptyInstanceID || slices.Contains(owned, newID) {
		return fmt.Errorf("new-ID restart did not return a distinct instance: %q", newID)
	}
	owned = append(owned, newID)
	if err := waitForBatch(ctx, c, newID, inputs[1]); err != nil {
		return err
	}
	stillOriginal, err := c.FetchOrchestrationMetadata(ctx, owned[1], api.WithFetchPayloads(true))
	if err != nil {
		return err
	}
	if preserved.ExecutionID == "" || stillOriginal.ExecutionID != preserved.ExecutionID ||
		stillOriginal.RuntimeStatus != api.RUNTIME_STATUS_COMPLETED ||
		stillOriginal.SerializedOutput != preserved.SerializedOutput {
		return errors.New("new-ID restart did not preserve the original completed execution")
	}

	resumedID := api.InstanceID(prefix + "suspend")
	owned = append(owned, resumedID)
	resumedInput := batchInput{BatchID: "resumed-batch", ItemCount: 40, WaitForRelease: true}
	if _, err := c.ScheduleNewOrchestration(ctx, workflowName,
		api.WithInstanceID(resumedID), api.WithInput(resumedInput)); err != nil {
		return err
	}
	if err := waitUntilReady(ctx, c, resumedID); err != nil {
		return err
	}
	if err := c.SuspendOrchestration(ctx, resumedID, "Go sample suspension"); err != nil {
		return err
	}
	if err := waitForStatus(ctx, c, resumedID, api.RUNTIME_STATUS_SUSPENDED); err != nil {
		return err
	}
	if err := c.RaiseEvent(ctx, resumedID, releaseEvent, api.WithEventPayload("process")); err != nil {
		return err
	}
	if err := remainsSuspended(ctx, c, resumedID, time.Second); err != nil {
		return err
	}
	if err := c.ResumeOrchestration(ctx, resumedID, "Go sample resumption"); err != nil {
		return err
	}
	if err := waitForBatch(ctx, c, resumedID, resumedInput); err != nil {
		return err
	}

	terminatedID := api.InstanceID(prefix + "terminate")
	owned = append(owned, terminatedID)
	if _, err := c.ScheduleNewOrchestration(ctx, workflowName,
		api.WithInstanceID(terminatedID),
		api.WithInput(batchInput{BatchID: "terminated-batch", ItemCount: 50, WaitForRelease: true})); err != nil {
		return err
	}
	if err := waitUntilReady(ctx, c, terminatedID); err != nil {
		return err
	}
	const reason = "terminated by Go management sample"
	if err := c.TerminateOrchestration(ctx, terminatedID,
		api.WithOutput(reason), api.WithRecursiveTerminate(false)); err != nil {
		return err
	}
	if err := waitForStatus(ctx, c, terminatedID, api.RUNTIME_STATUS_TERMINATED); err != nil {
		return err
	}
	terminated, err := c.FetchOrchestrationMetadata(ctx, terminatedID, api.WithFetchPayloads(true))
	if err != nil {
		return err
	}
	var terminationOutput string
	if err := terminated.ReadOutput(&terminationOutput); err != nil {
		return err
	}
	if terminationOutput != reason {
		return fmt.Errorf("termination output = %q, want %q", terminationOutput, reason)
	}

	// The service chooses the new restart ID, so query it separately by its exact ID prefix.
	groups := []struct {
		prefix string
		ids    []api.InstanceID
	}{
		{prefix, []api.InstanceID{owned[0], owned[1], owned[2], resumedID}},
		{string(newID), []api.InstanceID{newID}},
	}
	for _, group := range groups {
		query := api.OrchestrationQuery{
			InstanceIDPrefix: group.prefix,
			CreatedTimeFrom:  createdFrom,
			RuntimeStatus:    []api.OrchestrationStatus{api.RUNTIME_STATUS_COMPLETED},
			PageSize:         2,
		}
		if err := sample.Until(ctx, 100*time.Millisecond, func() (bool, error) {
			ids, err := queryOwned(ctx, c, query, group.ids)
			return len(ids) == len(group.ids), err
		}); err != nil {
			return fmt.Errorf("scoped completed-instance query did not return all owned IDs: %w", err)
		}
	}

	purgeCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if err := purgeOwned(purgeCtx, c, owned); err != nil {
		return err
	}
	for _, queryPrefix := range []string{prefix, string(newID)} {
		if err := sample.Until(purgeCtx, 100*time.Millisecond, func() (bool, error) {
			ids, err := queryOwned(purgeCtx, c, api.OrchestrationQuery{
				InstanceIDPrefix: queryPrefix, PageSize: 2,
			}, owned)
			return len(ids) == 0, err
		}); err != nil {
			return fmt.Errorf("purged IDs remain in the scoped query: %w", err)
		}
	}
	return nil
}

func waitForBatch(ctx context.Context, c *dts.Client, id api.InstanceID, input batchInput) error {
	var output batchResult
	if err := sample.Wait(ctx, c, id, &output); err != nil {
		return err
	}
	want := batchResult{BatchID: input.BatchID, ItemsProcessed: input.ItemCount, Status: "success"}
	return testutil.Require(output == want, "batch %s output = %+v, want %+v", id, output, want)
}

func waitForNewExecution(ctx context.Context, c metadataClient, id api.InstanceID, previous string) error {
	if previous == "" {
		return errors.New("restart cannot be verified: original execution ID is missing")
	}
	err := sample.Until(ctx, 100*time.Millisecond, func() (bool, error) {
		metadata, err := c.FetchOrchestrationMetadata(ctx, id)
		if errors.Is(err, api.ErrInstanceNotFound) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return metadata.ExecutionID != "" && metadata.ExecutionID != previous, nil
	})
	if err != nil {
		return fmt.Errorf("restart did not expose a new execution for %s: %w", id, err)
	}
	return nil
}

func waitUntilReady(ctx context.Context, c metadataClient, id api.InstanceID) error {
	return sample.Until(ctx, 100*time.Millisecond, func() (bool, error) {
		metadata, err := c.FetchOrchestrationMetadata(ctx, id, api.WithFetchPayloads(true))
		if errors.Is(err, api.ErrInstanceNotFound) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if metadata.IsComplete() {
			return false, fmt.Errorf("%s ended before its management gate: %s", id, metadata.RuntimeStatus)
		}
		if metadata.SerializedCustomStatus == "" {
			return false, nil
		}
		var state string
		if err := metadata.ReadCustomStatus(&state); err != nil {
			return false, err
		}
		return metadata.RuntimeStatus == api.RUNTIME_STATUS_RUNNING && state == waiting, nil
	})
}

func waitForStatus(ctx context.Context, c metadataClient, id api.InstanceID, status api.OrchestrationStatus) error {
	return sample.Until(ctx, 100*time.Millisecond, func() (bool, error) {
		metadata, err := c.FetchOrchestrationMetadata(ctx, id)
		if err != nil {
			return false, err
		}
		if metadata.RuntimeStatus == status {
			return true, nil
		}
		if metadata.IsComplete() {
			return false, fmt.Errorf("%s reached %s instead of %s", id, metadata.RuntimeStatus, status)
		}
		return false, nil
	})
}

func remainsSuspended(ctx context.Context, c metadataClient, id api.InstanceID, duration time.Duration) error {
	until := time.Now().Add(duration)
	return sample.Until(ctx, 100*time.Millisecond, func() (bool, error) {
		metadata, err := c.FetchOrchestrationMetadata(ctx, id)
		if err != nil {
			return false, err
		}
		if metadata.RuntimeStatus != api.RUNTIME_STATUS_SUSPENDED {
			return false, fmt.Errorf("%s processed work while suspended: %s", id, metadata.RuntimeStatus)
		}
		return !time.Now().Before(until), nil
	})
}

func queryOwned(ctx context.Context, c queryClient, query api.OrchestrationQuery, allowed []api.InstanceID) ([]api.InstanceID, error) {
	if query.InstanceIDPrefix == "" {
		return nil, errors.New("refusing an unscoped instance query")
	}
	var ids []api.InstanceID
	tokens := map[string]bool{}
	for {
		page, err := c.QueryInstances(ctx, query)
		if err != nil {
			return nil, err
		}
		if page == nil {
			return nil, errors.New("instance query returned a nil page")
		}
		for _, metadata := range page.Orchestrations {
			if metadata == nil || !slices.Contains(allowed, metadata.InstanceID) {
				return nil, errors.New("scoped query returned an instance not owned by this invocation")
			}
			if len(query.RuntimeStatus) > 0 && !slices.Contains(query.RuntimeStatus, metadata.RuntimeStatus) {
				return nil, fmt.Errorf("query returned unexpected status %s for %s", metadata.RuntimeStatus, metadata.InstanceID)
			}
			if !slices.Contains(ids, metadata.InstanceID) {
				ids = append(ids, metadata.InstanceID)
			}
		}
		if page.ContinuationToken == "" {
			return ids, nil
		}
		if tokens[page.ContinuationToken] {
			return nil, errors.New("instance query returned a repeated continuation token")
		}
		tokens[page.ContinuationToken] = true
		query.ContinuationToken = page.ContinuationToken
	}
}

func purgeOwned(ctx context.Context, c purgeClient, ids []api.InstanceID) error {
	if len(ids) == 0 {
		return errors.New("refusing to purge without exact owned instance IDs")
	}
	seen := map[api.InstanceID]bool{}
	for _, id := range ids {
		if id == api.EmptyInstanceID || seen[id] {
			return errors.New("purge IDs must be nonempty and unique")
		}
		seen[id] = true
	}
	result, err := c.PurgeInstances(ctx, api.PurgeInstancesRequest{
		InstanceIDs: slices.Clone(ids), Recursive: false,
	})
	if err != nil {
		return err
	}
	if result == nil || !result.IsComplete {
		return errors.New("exact-ID purge was not reported complete")
	}
	for _, id := range ids {
		err := sample.Until(ctx, 100*time.Millisecond, func() (bool, error) {
			_, err := c.FetchOrchestrationMetadata(ctx, id)
			if errors.Is(err, api.ErrInstanceNotFound) {
				return true, nil
			}
			return false, err
		})
		if err != nil {
			return fmt.Errorf("cannot verify exact-ID purge of %s; an acknowledged purge is not proof of deletion (target may be incompatible): %w", id, err)
		}
	}
	return nil
}
