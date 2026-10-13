package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
)

const (
	initialInterval = 5 * time.Second
	updatedInterval = 2 * time.Second
	scheduleLife    = 90 * time.Second
)

func run(ctx context.Context) (err error) {
	host, err := startWorker(ctx)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, host.Close()) }()

	id := string(sample.ID("scheduled-tasks"))
	handle, err := host.Client.ScheduledTasks().GetScheduleClient(id)
	if err != nil {
		return err
	}
	created := false
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if cleanupErr := removeSchedule(cleanupCtx, handle, created); cleanupErr != nil {
			err = errors.Join(err, fmt.Errorf("cleanup schedule %s: %w", id, cleanupErr))
		} else {
			fmt.Printf("Deleted schedule %s\n", id)
		}
	}()

	if err := handle.Create(ctx, creationOptions(id, time.Now().UTC())); err != nil {
		return err
	}
	created = true
	fmt.Printf("Schedule %s: westus reports every %s\n", id, initialInterval)
	if err := allowReports(ctx, 2*initialInterval+time.Second); err != nil {
		return err
	}
	if err := handle.Pause(ctx); err != nil {
		return err
	}
	fmt.Println("Paused schedule")

	interval := updatedInterval
	if err := handle.Update(ctx, dts.ScheduleUpdateOptions{
		Interval:                &interval,
		TypedOrchestrationInput: reportInput{ScheduleID: id, Phase: "updated", Region: "eastus"},
	}); err != nil {
		return err
	}
	if err := handle.Resume(ctx); err != nil {
		return err
	}
	fmt.Printf("Resumed schedule: eastus reports every %s\n", interval)
	return allowReports(ctx, 2*updatedInterval+time.Second)
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

func allowReports(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
