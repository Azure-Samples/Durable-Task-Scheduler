package main

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
	"github.com/microsoft/durabletask-go/task"
)

const (
	reportName      = "go-sample-schedules-report"
	activityName    = "go-sample-schedules-send-report"
	initialInterval = 5 * time.Second
	updatedInterval = 2 * time.Second
	scheduleLife    = 90 * time.Second
)

type reportInput struct {
	ScheduleID string `json:"schedule_id"`
	Phase      string `json:"phase"`
	Region     string `json:"region"`
}

type reportResult struct {
	ScheduleID string `json:"schedule_id"`
	Phase      string `json:"phase"`
	Message    string `json:"message"`
}

type observedReport struct {
	ID     api.InstanceID
	Status api.OrchestrationStatus
	Input  reportInput
}

type reportClient interface {
	QueryInstances(context.Context, api.OrchestrationQuery) (*api.OrchestrationQueryResult, error)
	FetchOrchestrationMetadata(context.Context, api.InstanceID, ...api.FetchOrchestrationMetadataOptions) (*api.OrchestrationMetadata, error)
}

type scheduleHandle interface {
	Describe(context.Context) (*dts.ScheduleDescription, error)
	Delete(context.Context) error
}

func main() {
	sample.Main("scheduled-tasks", run)
}

func run(ctx context.Context) (err error) {
	registry, err := newRegistry()
	if err != nil {
		return err
	}
	// Schedule deletion is durable work: its worker must outlive the run deadline.
	host, err := sample.Start(context.WithoutCancel(ctx), registry, nil, dts.WithScheduledTasks())
	if err != nil {
		return err
	}
	scheduleID := string(sample.ID("scheduled-tasks"))
	var handle *dts.ScheduleClient
	creationAttempted, creationConfirmed, deleted := false, false, false
	defer func() {
		if creationAttempted && !deleted {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if cleanupErr := removeSchedule(cleanupCtx, handle, creationConfirmed); cleanupErr != nil {
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
	deleted = true
	description, err = schedules.Get(ctx, scheduleID)
	if err != nil {
		return err
	}
	if description != nil {
		return fmt.Errorf("deleted schedule still exists: %+v", description)
	}
	fmt.Printf("Created/read/listed schedule %s\n", scheduleID)
	fmt.Printf("Verified initial recurring reports: %d (at least 2), Report for 'westus' generated\n", initialRuns)
	fmt.Println("Verified pause and sparse update: no updated runs during two intervals")
	fmt.Printf("Verified resumed reports: %d (at least 1), Report for 'eastus' generated\n", updatedRuns)
	fmt.Println("Deleted owned schedule; Describe reports not found and Get returns nil")
	return nil
}

func newRegistry() (*task.TaskRegistry, error) {
	registry := task.NewTaskRegistry()
	if err := registry.AddOrchestratorN(reportName, reportWorkflow); err != nil {
		return nil, err
	}
	if err := registry.AddActivityN(activityName, sendReport); err != nil {
		return nil, err
	}
	if err := dts.RegisterScheduledTasks(registry); err != nil {
		return nil, err
	}
	return registry, nil
}

func creationOptions(scheduleID string, now time.Time) dts.ScheduleCreationOptions {
	return dts.ScheduleCreationOptions{
		ScheduleID:              scheduleID,
		OrchestrationName:       reportName,
		TypedOrchestrationInput: reportInput{ScheduleID: scheduleID, Phase: "initial", Region: "westus"},
		Interval:                initialInterval,
		StartAt:                 now.Add(time.Second),
		EndAt:                   now.Add(scheduleLife),
		StartImmediatelyIfLate:  true,
	}
}

func reportWorkflow(ctx *task.OrchestrationContext) (any, error) {
	var input reportInput
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	var result reportResult
	if err := ctx.CallActivity(activityName, task.WithActivityInput(input)).Await(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func sendReport(ctx task.ActivityContext) (any, error) {
	var input reportInput
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	if input.ScheduleID == "" || input.Region == "" ||
		(input.Phase != "initial" && input.Phase != "updated") {
		return nil, errors.New("schedule_id, region, and a recognized phase are required")
	}
	return reportResult{
		ScheduleID: input.ScheduleID,
		Phase:      input.Phase,
		Message:    fmt.Sprintf("Report for '%s' generated", input.Region),
	}, nil
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

func removeSchedule(ctx context.Context, handle scheduleHandle, creationConfirmed bool) error {
	if !creationConfirmed {
		// Do not race a possibly queued Create with Delete of an absent entity.
		if err := sample.Until(ctx, 100*time.Millisecond, func() (bool, error) {
			_, err := handle.Describe(ctx)
			if errors.Is(err, dts.ErrScheduleNotFound) {
				return false, nil
			}
			return err == nil, err
		}); err != nil {
			return fmt.Errorf("schedule creation outcome is unknown; cleanup incomplete: %w", err)
		}
	}
	if err := handle.Delete(ctx); err != nil {
		return fmt.Errorf("delete owned recurring schedule: %w", err)
	}
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
