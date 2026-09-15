package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/exporthistory"
)

const cleanupTimeout = 30 * time.Second

type exportWorkerLifetime struct {
	context context.Context
	cancel  context.CancelFunc
}

func newExportWorkerLifetime(ctx context.Context) exportWorkerLifetime {
	workerCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	return exportWorkerLifetime{context: workerCtx, cancel: cancel}
}

func (w exportWorkerLifetime) close(closeHost func() error) error {
	defer w.cancel()
	return closeHost()
}

type cleanupJob interface {
	ID() string
	Delete(context.Context) error
	Describe(context.Context) (*exporthistory.ExportJobDescription, error)
}

func withJobCleanup(ctx, workerCtx context.Context, job cleanupJob, work func() error) (err error) {
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(workerCtx, cleanupTimeout)
		defer cancel()
		fmt.Printf("EXPORT_JOB_CLEANUP job_id=%s\n", job.ID())
		cleanupErr := deleteAndVerifyJob(cleanupCtx, job)
		if cleanupErr == nil {
			fmt.Printf("EXPORT_JOB_CLEANED job_id=%s\n", job.ID())
		}
		if err == nil {
			err = ctx.Err()
		}
		err = errors.Join(err, cleanupErr)
	}()
	return work()
}

func deleteAndVerifyJob(ctx context.Context, job cleanupJob) error {
	deleteErr := job.Delete(ctx)
	if deleteErr != nil {
		deleteErr = fmt.Errorf("delete this run's export job %s: %w", job.ID(), deleteErr)
	}
	// Delete can fail after clearing the entity but before purging the captured
	// generation. Always check absence, without treating it as a successful Delete.
	verifyErr := sample.Until(ctx, 250*time.Millisecond, func() (bool, error) {
		_, err := job.Describe(ctx)
		if errors.Is(err, exporthistory.ErrJobNotFound) {
			return true, nil
		}
		return false, err
	})
	if verifyErr != nil {
		verifyErr = fmt.Errorf("verify deletion of job %s: %w", job.ID(), verifyErr)
	}
	return errors.Join(deleteErr, verifyErr)
}

func pauseBeforeWrite(ctx context.Context, reportActive func()) func(context.Context) error {
	var once sync.Once
	return func(writeCtx context.Context) error {
		once.Do(reportActive)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-writeCtx.Done():
			return writeCtx.Err()
		}
	}
}
