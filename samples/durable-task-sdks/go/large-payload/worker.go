package main

import (
	"context"
	"fmt"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	"github.com/microsoft/durabletask-go/payload"
	"github.com/microsoft/durabletask-go/task"
)

const (
	thresholdBytes   = 64 * 1024
	grpcMessageBytes = 128 * 1024
	maxPayloadBytes  = 4 * 1024 * 1024
)

type payloadWorker struct {
	*sample.Host
	workflow  payloadWorkflow
	container string
}

func startWorker(ctx context.Context) (*payloadWorker, error) {
	container := string(sample.ID("large-payload"))
	workflow := newPayloadWorkflow(container)
	storeOptions, err := storageOptions(container)
	if err != nil {
		return nil, err
	}
	store, err := payload.NewAzureBlobStore(storeOptions)
	if err != nil {
		return nil, fmt.Errorf("configure payload store: %w", err)
	}
	options, err := sample.Options()
	if err != nil {
		return nil, err
	}
	// Share the store with the worker and client, keeping gRPC messages small.
	options.MaxSendMessageSize = grpcMessageBytes
	options.MaxReceiveMessageSize = grpcMessageBytes
	options.LargePayloads = &api.LargePayloadOptions{
		Store: store, Resolver: store,
		ThresholdBytes: thresholdBytes, MaxPayloadBytes: maxPayloadBytes,
	}
	registry := task.NewTaskRegistry()
	if err := workflow.register(registry); err != nil {
		return nil, err
	}
	host, err := sample.Start(ctx, registry, options)
	if err != nil {
		return nil, err
	}
	return &payloadWorker{Host: host, workflow: workflow, container: container}, nil
}

func (w payloadWorkflow) register(registry *task.TaskRegistry) error {
	if err := registry.AddOrchestratorN(w.orchestrator, w.orchestrate); err != nil {
		return err
	}
	if err := registry.AddActivityN(w.echo, echoData); err != nil {
		return err
	}
	return registry.AddActivityN(w.process, processData)
}
