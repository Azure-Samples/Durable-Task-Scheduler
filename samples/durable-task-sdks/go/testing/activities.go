package main

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/microsoft/durabletask-go/task"
)

const (
	validateName = "GoTestingValidate"
	chargeName   = "GoTestingCharge"
	shipName     = "GoTestingShip"
)

type shipment struct {
	Customer  string `json:"customer"`
	ItemCount int    `json:"itemCount"`
}

func validateOrder(input order) error {
	if strings.TrimSpace(input.Customer) == "" {
		return errors.New("order must have a customer name")
	}
	if len(input.Items) == 0 {
		return errors.New("order must contain at least one item")
	}
	_, err := totalCents(input.Items)
	return err
}

func totalCents(items []item) (int64, error) {
	var total int64
	for _, item := range items {
		if item.Quantity <= 0 || item.UnitPriceCents <= 0 {
			return 0, fmt.Errorf("invalid quantity or price for %q", item.Name)
		}
		if item.Quantity > math.MaxInt64/item.UnitPriceCents {
			return 0, errors.New("line total exceeds supported amount")
		}
		line := item.Quantity * item.UnitPriceCents
		if total > math.MaxInt64-line {
			return 0, errors.New("order total exceeds supported amount")
		}
		total += line
	}
	return total, nil
}

func chargePayment(amount int64) (string, error) {
	if amount <= 0 {
		return "", errors.New("payment amount must be positive")
	}
	// A deterministic stand-in for an idempotent payment gateway.
	return fmt.Sprintf("PAY-%d", amount), nil
}

func shipOrder(input shipment) (string, error) {
	if input.Customer == "" || input.ItemCount <= 0 {
		return "", errors.New("shipment requires a customer and items")
	}
	return fmt.Sprintf("TRACK-%s-%d", strings.ToUpper(input.Customer), input.ItemCount), nil
}

func validateActivity(ctx task.ActivityContext) (any, error) {
	var input order
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	return nil, validateOrder(input)
}

func chargeActivity(ctx task.ActivityContext) (any, error) {
	var amount int64
	if err := ctx.GetInput(&amount); err != nil {
		return nil, err
	}
	return chargePayment(amount)
}

func shipActivity(ctx task.ActivityContext) (any, error) {
	var input shipment
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	return shipOrder(input)
}
