package main

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
)

func TestScheduledTaskSystemRegistrations(t *testing.T) {
	registry, err := newRegistry()
	if err != nil {
		t.Fatal(err)
	}
	snapshot := registry.Snapshot()
	if len(snapshot.Entities) != 1 || snapshot.Entities[0] != "schedule" ||
		len(snapshot.Activities) != 1 || snapshot.Activities[0].Name != activityName ||
		len(snapshot.Orchestrators) != 3 {
		t.Fatalf("unexpected scheduled-task registry: %+v", snapshot)
	}
	for _, name := range []string{reportName, dts.ExecuteScheduleOperationOrchestratorName, dts.ExecuteScheduledTaskOrchestratorName} {
		found := false
		for _, registration := range snapshot.Orchestrators {
			if registration.Name == name {
				found = true
				if registration.Version != "" {
					t.Fatalf("system/sample orchestration %s unexpectedly versioned: %s", name, registration.Version)
				}
			}
		}
		if !found {
			t.Fatalf("missing required registration %s", name)
		}
	}
}

func TestScheduleOptionsAreBoundedAndOwnTargets(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	options := creationOptions("go-owned-schedule", now)
	if options.ScheduleID != "go-owned-schedule" || options.OrchestrationName != reportName ||
		options.Interval != 5*time.Second || !options.EndAt.Equal(now.Add(90*time.Second)) ||
		!options.EndAt.After(options.StartAt) || !options.StartImmediatelyIfLate {
		t.Fatalf("unexpected schedule options: %+v", options)
	}
	if options.RetryPolicy != nil || len(options.Tags) != 0 || len(options.ContextFields) != 0 || options.OrchestrationInstanceID != "" {
		t.Fatal("target prefix discovery requires direct, uniquely generated scheduled targets")
	}
}

type activityInput string

func (a activityInput) Context() context.Context { return context.Background() }
func (a activityInput) GetInput(target any) error {
	return json.Unmarshal([]byte(a), target)
}

func TestReportActivityAndVerification(t *testing.T) {
	got, err := sendReport(activityInput(`{"schedule_id":"go-owned","phase":"initial","region":"westus"}`))
	want := reportResult{ScheduleID: "go-owned", Phase: "initial", Message: "Report for 'westus' generated"}
	if err != nil || got != want {
		t.Fatalf("report = %+v, %v", got, err)
	}
	input := reportInput{ScheduleID: "go-owned", Phase: "initial", Region: "westus"}
	if err := checkReport(want, input); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []reportResult{
		{ScheduleID: "someone-else", Phase: want.Phase, Message: want.Message},
		{ScheduleID: want.ScheduleID, Phase: "updated", Message: want.Message},
		{ScheduleID: want.ScheduleID, Phase: want.Phase, Message: "not generated"},
	} {
		if err := checkReport(bad, input); err == nil {
			t.Fatalf("wrong report accepted: %+v", bad)
		}
	}
	if _, err := sendReport(activityInput(`{}`)); err == nil {
		t.Fatal("empty scheduled report input was accepted")
	}
}

func TestDescriptionMustReflectUpdate(t *testing.T) {
	input := reportInput{ScheduleID: "go-owned", Phase: "updated", Region: "eastus"}
	payload, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	description := &dts.ScheduleDescription{
		ScheduleID: input.ScheduleID, OrchestrationName: reportName,
		OrchestrationInput: string(payload), Status: dts.ScheduleStatusPaused, Interval: updatedInterval,
	}
	if err := checkDescription(description, input.ScheduleID, dts.ScheduleStatusPaused, updatedInterval, input); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*dts.ScheduleDescription){
		func(d *dts.ScheduleDescription) { d.Interval = initialInterval },
		func(d *dts.ScheduleDescription) { d.Status = dts.ScheduleStatusActive },
		func(d *dts.ScheduleDescription) { d.OrchestrationInput = `{}` },
		func(d *dts.ScheduleDescription) { d.ScheduleID = "someone-else" },
	} {
		invalid := *description
		mutate(&invalid)
		if err := checkDescription(&invalid, input.ScheduleID, dts.ScheduleStatusPaused, updatedInterval, input); err == nil {
			t.Fatal("stale/incorrect schedule description passed")
		}
	}
}

type fakeSchedule struct {
	describes   int
	appearAfter int
	deleteWorks bool
	deleted     bool
	calls       []string
}

func (f *fakeSchedule) Describe(context.Context) (*dts.ScheduleDescription, error) {
	f.describes++
	f.calls = append(f.calls, "describe")
	if f.describes <= f.appearAfter || f.deleted && f.deleteWorks {
		return nil, dts.ErrScheduleNotFound
	}
	return &dts.ScheduleDescription{}, nil
}

func (f *fakeSchedule) Delete(context.Context) error {
	f.calls = append(f.calls, "delete")
	f.deleted = true
	return nil
}

func TestCleanupWaitsForAcceptedCreationAndVerifiesDelete(t *testing.T) {
	fake := &fakeSchedule{appearAfter: 1, deleteWorks: true}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := removeSchedule(ctx, fake, false); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(fake.calls, []string{"describe", "describe", "delete", "describe"}) {
		t.Fatalf("cleanup raced creation or omitted verification: %v", fake.calls)
	}
}

func TestCleanupNeverTreatsAcknowledgementAsAbsence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	fake := &fakeSchedule{}
	if err := removeSchedule(ctx, fake, true); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ineffective delete must fail: %v", err)
	}
}

func TestUnknownCreationDoesNotRaceDeletion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	fake := &fakeSchedule{appearAfter: 1000}
	if err := removeSchedule(ctx, fake, false); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("unknown creation must report incomplete cleanup: %v", err)
	}
	if fake.deleted {
		t.Fatal("deleted before establishing whether the accepted create completed")
	}
}

type fakeReports struct {
	page     *api.OrchestrationQueryResult
	metadata *api.OrchestrationMetadata
	fetches  int
}

func (f *fakeReports) QueryInstances(context.Context, api.OrchestrationQuery) (*api.OrchestrationQueryResult, error) {
	return f.page, nil
}

func (f *fakeReports) FetchOrchestrationMetadata(context.Context, api.InstanceID, ...api.FetchOrchestrationMetadataOptions) (*api.OrchestrationMetadata, error) {
	f.fetches++
	return f.metadata, nil
}

func TestReportsVerifyServerOutputAndDoNotFetchForeignWork(t *testing.T) {
	metadata := &api.OrchestrationMetadata{
		InstanceID: "go-owned-tick", Name: reportName, RuntimeStatus: api.RUNTIME_STATUS_COMPLETED,
		SerializedInput:  `{"schedule_id":"go-owned","phase":"initial","region":"westus"}`,
		SerializedOutput: `{"schedule_id":"go-owned","phase":"initial","message":"Report for 'westus' generated"}`,
	}
	fake := &fakeReports{
		page:     &api.OrchestrationQueryResult{Orchestrations: []*api.OrchestrationMetadata{metadata}},
		metadata: metadata,
	}
	reports, err := readReports(context.Background(), fake, "go-owned")
	if err != nil || len(reports) != 1 || reports[0].Status != api.RUNTIME_STATUS_COMPLETED {
		t.Fatalf("observed reports = %+v, %v", reports, err)
	}
	metadata.SerializedOutput = `{"message":"wrong"}`
	if _, err := readReports(context.Background(), fake, "go-owned"); err == nil {
		t.Fatal("wrong server output was accepted")
	}
	metadata.InstanceID = "someone-else"
	fake.fetches = 0
	if _, err := readReports(context.Background(), fake, "go-owned"); err == nil || fake.fetches != 0 {
		t.Fatalf("foreign work was accepted or fetched: err=%v fetches=%d", err, fake.fetches)
	}
}

func TestReportPollingRequiresDistinctCompletedInstances(t *testing.T) {
	metadata := &api.OrchestrationMetadata{
		InstanceID: "go-owned-tick", Name: reportName, RuntimeStatus: api.RUNTIME_STATUS_RUNNING,
		SerializedInput:  `{"schedule_id":"go-owned","phase":"initial","region":"westus"}`,
		SerializedOutput: `{"schedule_id":"go-owned","phase":"initial","message":"Report for 'westus' generated"}`,
	}
	fake := &fakeReports{
		page:     &api.OrchestrationQueryResult{Orchestrations: []*api.OrchestrationMetadata{metadata, metadata}},
		metadata: metadata,
	}
	for _, test := range []struct {
		name    string
		status  api.OrchestrationStatus
		minimum int
		want    int
	}{
		{"not completed", api.RUNTIME_STATUS_RUNNING, 1, 0},
		{"duplicate is not a second tick", api.RUNTIME_STATUS_COMPLETED, 2, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			metadata.RuntimeStatus = test.status
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()
			count, err := waitForReports(ctx, fake, "go-owned", "initial", test.minimum)
			if count != test.want || !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("count = %d, error = %v", count, err)
			}
		})
	}
}
