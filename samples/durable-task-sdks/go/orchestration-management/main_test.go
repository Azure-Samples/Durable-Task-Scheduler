package main

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/microsoft/durabletask-go/api"
)

type fakeManagementClient struct {
	pages       []*api.OrchestrationQueryResult
	queries     []api.OrchestrationQuery
	metadata    *api.OrchestrationMetadata
	fetchError  error
	purge       api.PurgeInstancesRequest
	deleteWorks bool
}

func (f *fakeManagementClient) QueryInstances(_ context.Context, query api.OrchestrationQuery) (*api.OrchestrationQueryResult, error) {
	f.queries = append(f.queries, query)
	if len(f.pages) == 0 {
		return &api.OrchestrationQueryResult{}, nil
	}
	page := f.pages[0]
	f.pages = f.pages[1:]
	return page, nil
}

func (f *fakeManagementClient) FetchOrchestrationMetadata(context.Context, api.InstanceID, ...api.FetchOrchestrationMetadataOptions) (*api.OrchestrationMetadata, error) {
	return f.metadata, f.fetchError
}

func (f *fakeManagementClient) PurgeInstances(_ context.Context, request api.PurgeInstancesRequest) (*api.PurgeInstancesResult, error) {
	f.purge = request
	if f.deleteWorks {
		f.fetchError = api.ErrInstanceNotFound
	}
	return &api.PurgeInstancesResult{IsComplete: true, DeletedInstanceCount: len(request.InstanceIDs)}, nil
}

func TestScopedQueryPagination(t *testing.T) {
	a, b := api.InstanceID("go-owned-a"), api.InstanceID("go-owned-b")
	fake := &fakeManagementClient{pages: []*api.OrchestrationQueryResult{
		{Orchestrations: []*api.OrchestrationMetadata{{InstanceID: a}}, ContinuationToken: "page-2"},
		{Orchestrations: []*api.OrchestrationMetadata{{InstanceID: b}}},
	}}
	got, err := queryOwned(context.Background(), fake, api.OrchestrationQuery{
		InstanceIDPrefix: "go-owned-", PageSize: 1,
	}, []api.InstanceID{a, b})
	if err != nil || !reflect.DeepEqual(got, []api.InstanceID{a, b}) {
		t.Fatalf("query = %v, %v", got, err)
	}
	if len(fake.queries) != 2 || fake.queries[1].ContinuationToken != "page-2" ||
		fake.queries[1].InstanceIDPrefix != "go-owned-" {
		t.Fatalf("pagination lost scope: %+v", fake.queries)
	}
}

func TestQueryRejectsUnsafeOrBrokenResults(t *testing.T) {
	for _, test := range []struct {
		name  string
		query api.OrchestrationQuery
		pages []*api.OrchestrationQueryResult
	}{
		{"unscoped", api.OrchestrationQuery{}, nil},
		{"unrelated ID", api.OrchestrationQuery{InstanceIDPrefix: "go-owned-"}, []*api.OrchestrationQueryResult{
			{Orchestrations: []*api.OrchestrationMetadata{{InstanceID: "someone-else"}}},
		}},
		{"repeated token", api.OrchestrationQuery{InstanceIDPrefix: "go-owned-"}, []*api.OrchestrationQueryResult{
			{ContinuationToken: "same"}, {ContinuationToken: "same"},
		}},
		{"wrong status", api.OrchestrationQuery{
			InstanceIDPrefix: "go-owned-", RuntimeStatus: []api.OrchestrationStatus{api.RUNTIME_STATUS_COMPLETED},
		}, []*api.OrchestrationQueryResult{
			{Orchestrations: []*api.OrchestrationMetadata{{InstanceID: "go-owned-a", RuntimeStatus: api.RUNTIME_STATUS_RUNNING}}},
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeManagementClient{pages: test.pages}
			if _, err := queryOwned(context.Background(), fake, test.query, []api.InstanceID{"go-owned-a"}); err == nil {
				t.Fatal("invalid query passed")
			}
		})
	}
}

func TestPurgeUsesExactIDsAndVerifiesDeletion(t *testing.T) {
	ids := []api.InstanceID{"go-owned-a", "go-owned-b"}
	fake := &fakeManagementClient{deleteWorks: true}
	if err := purgeOwned(context.Background(), fake, ids); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fake.purge.InstanceIDs, ids) || fake.purge.Filter != nil || fake.purge.Recursive {
		t.Fatalf("unsafe purge request: %+v", fake.purge)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	falseSuccess := &fakeManagementClient{metadata: &api.OrchestrationMetadata{InstanceID: ids[0]}}
	if err := purgeOwned(ctx, falseSuccess, ids); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("acknowledged but ineffective purge must fail: %v", err)
	}
	for _, invalid := range [][]api.InstanceID{nil, {""}, {"a", "a"}} {
		if err := purgeOwned(context.Background(), fake, invalid); err == nil {
			t.Fatalf("invalid purge IDs accepted: %v", invalid)
		}
	}
}

func TestRestartMustObserveNewExecution(t *testing.T) {
	fake := &fakeManagementClient{metadata: &api.OrchestrationMetadata{
		ExecutionID: "old", RuntimeStatus: api.RUNTIME_STATUS_COMPLETED,
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := waitForNewExecution(ctx, fake, "go-owned-a", "old"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("old completed execution was accepted as a restart: %v", err)
	}
	fake.metadata.ExecutionID = "new"
	if err := waitForNewExecution(context.Background(), fake, "go-owned-a", "old"); err != nil {
		t.Fatal(err)
	}
	if err := waitForNewExecution(context.Background(), fake, "go-owned-a", ""); err == nil {
		t.Fatal("missing original execution ID was accepted")
	}
}

func TestSuspensionCannotPassWithCompletedInstance(t *testing.T) {
	fake := &fakeManagementClient{metadata: &api.OrchestrationMetadata{
		InstanceID: "go-owned-a", RuntimeStatus: api.RUNTIME_STATUS_COMPLETED,
	}}
	if err := remainsSuspended(context.Background(), fake, "go-owned-a", 0); err == nil {
		t.Fatal("a completed instance was considered suspended")
	}
	if err := waitForStatus(context.Background(), fake, "go-owned-a", api.RUNTIME_STATUS_TERMINATED); err == nil {
		t.Fatal("completion was considered termination")
	}
}

type activityInput string

func (a activityInput) Context() context.Context { return context.Background() }
func (a activityInput) GetInput(target any) error {
	return json.Unmarshal([]byte(a), target)
}

func TestBatchActivity(t *testing.T) {
	got, err := processBatch(activityInput(`{"batch_id":"batch-1","item_count":10}`))
	if err != nil || got != (batchResult{BatchID: "batch-1", ItemsProcessed: 10, Status: "success"}) {
		t.Fatalf("batch result = %+v, %v", got, err)
	}
	for _, input := range []string{`{}`, `{"batch_id":"bad","item_count":-1}`, `{"item_count":"ten"}`} {
		if _, err := processBatch(activityInput(input)); err == nil {
			t.Fatalf("invalid batch accepted: %s", input)
		}
	}
}
