package main

import (
	"errors"
	"strings"

	"github.com/microsoft/durabletask-go/task"
	"go.opentelemetry.io/otel/trace"
)

const (
	validateOrderName    = "GoOpenTelemetryTracingValidateOrder"
	processPaymentName   = "GoOpenTelemetryTracingProcessPayment"
	shipOrderName        = "GoOpenTelemetryTracingShipOrder"
	sendNotificationName = "GoOpenTelemetryTracingSendNotification"
)

func validateOrder(ctx task.ActivityContext) (any, error) {
	return orderStage(ctx, "Validated")
}

func processPayment(ctx task.ActivityContext) (any, error) {
	return orderStage(ctx, "Paid")
}

func shipOrder(ctx task.ActivityContext) (any, error) {
	return orderStage(ctx, "Shipped")
}

func sendNotification(tracer trace.Tracer, targetURL string) task.Activity {
	return func(ctx task.ActivityContext) (any, error) {
		result, err := orderStage(ctx, "Notified")
		if err != nil {
			return nil, err
		}
		if err := callNotification(ctx.Context(), tracer, targetURL); err != nil {
			return nil, err
		}
		return result, nil
	}
}

func orderStage(ctx task.ActivityContext, stage string) (string, error) {
	var order string
	if err := ctx.GetInput(&order); err != nil {
		return "", err
	}
	if strings.TrimSpace(order) == "" {
		return "", errors.New("order must not be empty")
	}
	return stage + "(" + order + ")", nil
}
