package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
)

type scheduleHandle interface {
	Describe(context.Context) (*dts.ScheduleDescription, error)
	Delete(context.Context) error
}

func removeSchedule(ctx context.Context, handle scheduleHandle, creationConfirmed bool) error {
	if !creationConfirmed {
		// Do not let Delete overtake a Create that was accepted before a timeout.
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
	return nil
}
