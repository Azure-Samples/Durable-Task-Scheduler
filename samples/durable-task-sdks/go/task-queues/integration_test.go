package main

import (
	"context"
	"testing"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/testutil"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
)

func TestIntegration(t *testing.T) {
	ctx := testutil.IntegrationContext(t)
	err := withWorker(ctx, func(ctx context.Context, c *dts.Client) error {
		id, _, err := executeOrder(ctx, c, order{ID: "integration-order"})
		if err != nil {
			return err
		}
		metadata, err := c.FetchOrchestrationMetadata(ctx, id)
		if err != nil {
			return err
		}
		if err := testutil.Require(metadata.Tags[queueTag] == "orders",
			"parent queue tags = %v", metadata.Tags); err != nil {
			return err
		}
		history, err := c.GetOrchestrationHistory(ctx, id, api.HistoryQuery{MaxEvents: 100})
		if err != nil {
			return err
		}
		foundAccept, foundPartner, foundChild := false, false, false
		for _, event := range history.Events {
			if scheduled := event.TaskScheduled; scheduled != nil {
				switch scheduled.Name {
				case acceptName:
					foundAccept = scheduled.Tags[queueTag] == "orders"
				case partnerName:
					foundPartner = scheduled.Tags[queueTag] == "partner-api"
				}
			}
			if child := event.SubOrchestrationInstanceCreated; child != nil {
				foundChild = child.Name == childName && child.Tags[queueTag] == "partner-api"
			}
		}
		if err := testutil.Require(foundAccept && foundPartner && foundChild,
			"history routing tags: inherited=%t, activity override=%t, child override=%t",
			foundAccept, foundPartner, foundChild); err != nil {
			return err
		}
		childID := api.InstanceID(string(id) + "-partner")
		child, err := c.FetchOrchestrationMetadata(ctx, childID)
		if err != nil {
			return err
		}
		if err := testutil.Require(child.RuntimeStatus == api.RUNTIME_STATUS_COMPLETED &&
			child.Tags[queueTag] == "partner-api",
			"child status = %s, tags = %v", child.RuntimeStatus, child.Tags); err != nil {
			return err
		}
		childHistory, err := c.GetOrchestrationHistory(ctx, childID, api.HistoryQuery{MaxEvents: 100})
		if err != nil {
			return err
		}
		for _, event := range childHistory.Events {
			if scheduled := event.TaskScheduled; scheduled != nil && scheduled.Name == partnerName {
				return testutil.Require(scheduled.Tags[queueTag] == "partner-api",
					"child activity inherited tags = %v", scheduled.Tags)
			}
		}
		return testutil.Require(false, "child activity missing from history")
	})
	if err != nil {
		t.Fatal(err)
	}
}
