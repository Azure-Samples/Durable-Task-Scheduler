package main

import (
	"context"
	"errors"
	"time"

	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
)

func stopOwned(c *dts.Client, ids []api.InstanceID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var cleanupErr error
	for _, id := range ids {
		metadata, err := c.FetchOrchestrationMetadata(ctx, id)
		if errors.Is(err, api.ErrInstanceNotFound) {
			continue
		}
		if err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
			continue
		}
		if metadata.IsComplete() {
			continue
		}
		if err := c.TerminateOrchestration(ctx, id, api.WithRecursiveTerminate(false)); err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
			continue
		}
		_, err = c.WaitForOrchestrationCompletion(ctx, id)
		cleanupErr = errors.Join(cleanupErr, err)
	}
	return cleanupErr
}
