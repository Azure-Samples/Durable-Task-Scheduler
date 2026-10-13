package main

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/testutil"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
)

type observedReport struct {
	ID     api.InstanceID
	Status api.OrchestrationStatus
	Input  reportInput
}

type reportClient interface {
	QueryInstances(context.Context, api.OrchestrationQuery) (*api.OrchestrationQueryResult, error)
	FetchOrchestrationMetadata(context.Context, api.InstanceID, ...api.FetchOrchestrationMetadataOptions) (*api.OrchestrationMetadata, error)
}

func TestIntegration(t *testing.T) {
	ctx := testutil.IntegrationContext(t)
	if err := verifySchedules(ctx); err != nil {
		t.Fatal(err)
	}
}

func verifySchedules(ctx context.Context) (err error) {
	host, err := startWorker(ctx)
	if err != nil {
		return err
	}
	scheduleID := string(sample.ID("scheduled-tasks"))
	var handle *dts.ScheduleClient
	creationAttempted, creationConfirmed, deleted := false, false, false
	defer func() {
		if creationAttempted && !deleted {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			cleanupErr := removeSchedule(cleanupCtx, handle, creationConfirmed)
			if cleanupErr == nil {
				cleanupErr = waitForScheduleDeletion(cleanupCtx, handle)
			}
			if cleanupErr != nil {
				err = errors.Join(err, fmt.Errorf("cleanup schedule %s: %w", scheduleID, cleanupErr))
			}
			cancel()
		}
		err = errors.Join(err, host.Close())
	}()

	schedules := host.Client.ScheduledTasks()
	// Retain the handle before Create: the server may accept work before a wait times out.
	handle, err = schedules.GetScheduleClient(scheduleID)
	if err != nil {
		return err
	}
	options := creationOptions(scheduleID, time.Now().UTC())
	creationAttempted = true
	if err := handle.Create(ctx, options); err != nil {
		return err
	}
	creationConfirmed = true
	initialInput := reportInput{ScheduleID: scheduleID, Phase: "initial", Region: "westus"}
	description, err := schedules.Get(ctx, scheduleID)
	if err != nil {
		return err
	}
	if err := checkDescription(description, scheduleID, dts.ScheduleStatusActive, initialInterval, initialInput); err != nil {
		return err
	}
	if err := waitUntilListed(ctx, schedules, scheduleID); err != nil {
		return err
	}
	initialRuns, err := waitForReports(ctx, host.Client, scheduleID, "initial", 2)
	if err != nil {
		return err
	}
	if err := handle.Pause(ctx); err != nil {
		return err
	}
	paused, err := handle.Describe(ctx)
	if err != nil {
		return err
	}
	if err := checkDescription(paused, scheduleID, dts.ScheduleStatusPaused, initialInterval, initialInput); err != nil {
		return err
	}
	if paused.LastRunAt.IsZero() || !paused.NextRunAt.IsZero() {
		return fmt.Errorf("paused schedule has invalid run timestamps: last=%s next=%s", paused.LastRunAt, paused.NextRunAt)
	}

	updatedInput := reportInput{ScheduleID: scheduleID, Phase: "updated", Region: "eastus"}
	interval := updatedInterval
	start := time.Now().UTC().Add(time.Second)
	if err := handle.Update(ctx, dts.ScheduleUpdateOptions{
		TypedOrchestrationInput: updatedInput,
		Interval:                &interval,
		StartAt:                 &start,
	}); err != nil {
		return err
	}
	updated, err := handle.Describe(ctx)
	if err != nil {
		return err
	}
	if err := checkDescription(updated, scheduleID, dts.ScheduleStatusPaused, updatedInterval, updatedInput); err != nil {
		return err
	}
	if !updated.EndAt.Equal(options.EndAt) {
		return errors.New("sparse schedule update did not preserve the finite end time")
	}
	quietUntil := time.Now().Add(2 * updatedInterval)
	if err := sample.Until(ctx, 150*time.Millisecond, func() (bool, error) {
		description, err := handle.Describe(ctx)
		if err != nil {
			return false, err
		}
		if err := checkDescription(description, scheduleID, dts.ScheduleStatusPaused, updatedInterval, updatedInput); err != nil {
			return false, err
		}
		if !description.LastRunAt.Equal(paused.LastRunAt) || !description.NextRunAt.IsZero() {
			return false, errors.New("schedule advanced while paused")
		}
		reports, err := readReports(ctx, host.Client, scheduleID)
		if err != nil {
			return false, err
		}
		for _, report := range reports {
			if report.Input.Phase == "updated" {
				return false, fmt.Errorf("updated report %s started while the schedule was paused", report.ID)
			}
		}
		return !time.Now().Before(quietUntil), nil
	}); err != nil {
		return err
	}

	if err := handle.Resume(ctx); err != nil {
		return err
	}
	resumed, err := handle.Describe(ctx)
	if err != nil {
		return err
	}
	if err := checkDescription(resumed, scheduleID, dts.ScheduleStatusActive, updatedInterval, updatedInput); err != nil {
		return err
	}
	updatedRuns, err := waitForReports(ctx, host.Client, scheduleID, "updated", 1)
	if err != nil {
		return err
	}
	if err := removeSchedule(ctx, handle, true); err != nil {
		return err
	}
	if err := waitForScheduleDeletion(ctx, handle); err != nil {
		return err
	}
	deleted = true
	description, err = schedules.Get(ctx, scheduleID)
	if err != nil {
		return err
	}
	if description != nil {
		return fmt.Errorf("deleted schedule still exists: %+v", description)
	}
	return testutil.Require(initialRuns >= 2 && updatedRuns >= 1,
		"completed reports: initial=%d updated=%d, want at least 2 and 1", initialRuns, updatedRuns)
}

func checkDescription(description *dts.ScheduleDescription, id string, status dts.ScheduleStatus, interval time.Duration, input reportInput) error {
	if description == nil {
		return errors.New("schedule description is missing")
	}
	if description.ScheduleID != id || description.OrchestrationName != reportName ||
		description.Status != status || description.Interval != interval {
		return fmt.Errorf("unexpected schedule configuration: %+v", description)
	}
	var stored reportInput
	if err := description.ReadInput(&stored); err != nil {
		return err
	}
	if stored != input {
		return fmt.Errorf("stored schedule input = %+v, want %+v", stored, input)
	}
	return nil
}

func waitUntilListed(ctx context.Context, schedules *dts.ScheduledTaskClient, id string) error {
	return sample.Until(ctx, 150*time.Millisecond, func() (bool, error) {
		query := dts.ScheduleQuery{ScheduleIDPrefix: id, PageSize: 5}
		tokens := map[string]bool{}
		found := false
		for {
			page, err := schedules.List(ctx, query)
			if err != nil {
				return false, err
			}
			if page == nil {
				return false, errors.New("schedule list returned a nil page")
			}
			for _, description := range page.Schedules {
				if description == nil || description.ScheduleID != id {
					return false, errors.New("schedule list returned an ID outside this invocation")
				}
				found = true
			}
			if page.ContinuationToken == "" {
				return found, nil
			}
			if tokens[page.ContinuationToken] {
				return false, errors.New("schedule list returned a repeated continuation token")
			}
			tokens[page.ContinuationToken] = true
			query.ContinuationToken = page.ContinuationToken
		}
	})
}

func waitForReports(ctx context.Context, c reportClient, scheduleID, phase string, minimum int) (int, error) {
	count := 0
	err := sample.Until(ctx, 150*time.Millisecond, func() (bool, error) {
		reports, err := readReports(ctx, c, scheduleID)
		if err != nil {
			return false, err
		}
		count = 0
		for _, report := range reports {
			if report.Input.Phase == phase && report.Status == api.RUNTIME_STATUS_COMPLETED {
				count++
			}
		}
		return count >= minimum, nil
	})
	if err != nil {
		return count, fmt.Errorf("verify %s scheduled reports: observed %d completed, want at least %d: %w", phase, count, minimum, err)
	}
	return count, nil
}

func readReports(ctx context.Context, c reportClient, scheduleID string) ([]observedReport, error) {
	if scheduleID == "" {
		return nil, errors.New("refusing an unscoped scheduled-report query")
	}
	// With no retry/tags/context wrapper, the published SDK creates targets with this prefix.
	query := api.OrchestrationQuery{InstanceIDPrefix: scheduleID + "-", PageSize: 25}
	var reports []observedReport
	var seen []api.InstanceID
	tokens := map[string]bool{}
	for {
		page, err := c.QueryInstances(ctx, query)
		if err != nil {
			return nil, err
		}
		if page == nil {
			return nil, errors.New("report query returned a nil page")
		}
		for _, metadata := range page.Orchestrations {
			if metadata == nil || !strings.HasPrefix(string(metadata.InstanceID), query.InstanceIDPrefix) ||
				metadata.Name != reportName {
				return nil, errors.New("report query returned work outside this invocation")
			}
			if slices.Contains(seen, metadata.InstanceID) {
				continue
			}
			seen = append(seen, metadata.InstanceID)
			full, err := c.FetchOrchestrationMetadata(ctx, metadata.InstanceID, api.WithFetchPayloads(true))
			if err != nil {
				return nil, err
			}
			var input reportInput
			if err := full.ReadInput(&input); err != nil {
				return nil, err
			}
			if input.ScheduleID != scheduleID ||
				!(input.Phase == "initial" && input.Region == "westus" || input.Phase == "updated" && input.Region == "eastus") {
				return nil, fmt.Errorf("unexpected scheduled input for %s: %+v", full.InstanceID, input)
			}
			if full.IsComplete() {
				if full.RuntimeStatus != api.RUNTIME_STATUS_COMPLETED {
					return nil, fmt.Errorf("scheduled report %s ended with %s: %+v", full.InstanceID, full.RuntimeStatus, full.FailureDetails)
				}
				var output reportResult
				if err := full.ReadOutput(&output); err != nil {
					return nil, err
				}
				if err := checkReport(output, input); err != nil {
					return nil, err
				}
			}
			reports = append(reports, observedReport{ID: full.InstanceID, Status: full.RuntimeStatus, Input: input})
		}
		if page.ContinuationToken == "" {
			return reports, nil
		}
		if tokens[page.ContinuationToken] {
			return nil, errors.New("report query returned a repeated continuation token")
		}
		tokens[page.ContinuationToken] = true
		query.ContinuationToken = page.ContinuationToken
	}
}

func checkReport(result reportResult, input reportInput) error {
	want := reportResult{
		ScheduleID: input.ScheduleID, Phase: input.Phase,
		Message: fmt.Sprintf("Report for '%s' generated", input.Region),
	}
	if result != want {
		return fmt.Errorf("scheduled report = %+v, want %+v", result, want)
	}
	return nil
}

func waitForScheduleDeletion(ctx context.Context, handle scheduleHandle) error {
	if err := sample.Until(ctx, 100*time.Millisecond, func() (bool, error) {
		_, err := handle.Describe(ctx)
		if errors.Is(err, dts.ErrScheduleNotFound) {
			return true, nil
		}
		return false, err
	}); err != nil {
		return fmt.Errorf("schedule deletion was acknowledged but absence could not be verified: %w", err)
	}
	return nil
}
