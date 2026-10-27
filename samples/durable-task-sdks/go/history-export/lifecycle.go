package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const cleanupTimeout = 30 * time.Second

type cleanupJob interface {
	ID() string
	Delete(context.Context) error
}

func withJobCleanup(ctx, workerCtx context.Context, job cleanupJob, work func() error) (err error) {
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(workerCtx, cleanupTimeout)
		defer cancel()
		cleanupErr := job.Delete(cleanupCtx)
		if err == nil {
			err = ctx.Err()
		}
		if cleanupErr != nil {
			cleanupErr = fmt.Errorf("delete export job %s: %w", job.ID(), cleanupErr)
		}
		err = errors.Join(err, cleanupErr)
	}()
	return work()
}
