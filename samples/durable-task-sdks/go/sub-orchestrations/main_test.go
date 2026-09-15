package main

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/microsoft/durabletask-go/task"
)

type activityInput []byte

func (input activityInput) GetInput(target any) error { return json.Unmarshal(input, target) }
func (activityInput) Context() context.Context        { return context.Background() }

func TestOrderOutcomesAndShortCircuiting(t *testing.T) {
	activities := map[string]task.Activity{
		inventoryName: checkInventory, paymentName: chargePayment, shippingName: shipOrder, notificationName: notifyCustomer,
	}
	for _, scenario := range []struct {
		failAt string
		reason string
		steps  []string
		calls  []string
	}{
		{"", "", []string{"inventory", "payment", "shipping", "notification"},
			[]string{inventoryName, paymentName, shippingName, notificationName}},
		{"inventory", "out of stock", []string{"inventory"}, []string{inventoryName}},
		{"payment", "payment failed", []string{"inventory", "payment"}, []string{inventoryName, paymentName}},
		{"shipping", "shipping failed", []string{"inventory", "payment", "shipping"},
			[]string{inventoryName, paymentName, shippingName}},
		{"notification", "customer notification failed", []string{"inventory", "payment", "shipping", "notification"},
			[]string{inventoryName, paymentName, shippingName, notificationName}},
	} {
		t.Run("failure-at-"+scenario.failAt, func(t *testing.T) {
			var calls []string
			result, err := processOrder(Order{ID: "order-1", FailAt: scenario.failAt}, func(name string, input Order) (bool, error) {
				calls = append(calls, name)
				data, err := json.Marshal(input)
				if err != nil {
					return false, err
				}
				output, err := activities[name](activityInput(data))
				if err != nil {
					return false, err
				}
				return output.(bool), nil
			})
			if err != nil {
				t.Fatal(err)
			}
			status := "failed"
			if scenario.failAt == "" {
				status = "completed"
			}
			want := OrderResult{Order: "order-1", Status: status, Reason: scenario.reason, Steps: scenario.steps}
			if !reflect.DeepEqual(result, want) || !reflect.DeepEqual(calls, scenario.calls) {
				t.Fatalf("result = %+v, calls = %v; want %+v, %v", result, calls, want, scenario.calls)
			}
		})
	}
}

func TestUnexpectedActivityFailurePropagates(t *testing.T) {
	failure := errors.New("payment service unavailable")
	var calls []string
	_, err := processOrder(Order{ID: "order-1"}, func(name string, _ Order) (bool, error) {
		calls = append(calls, name)
		if name == paymentName {
			return false, failure
		}
		return true, nil
	})
	if !errors.Is(err, failure) || !reflect.DeepEqual(calls, []string{inventoryName, paymentName}) {
		t.Fatalf("failure = %v, calls = %v", err, calls)
	}
}

func TestFixtureAndValidation(t *testing.T) {
	output, err := integrationOrders(nil)
	if err != nil {
		t.Fatal(err)
	}
	orders := output.([]Order)
	if len(orders) != 5 {
		t.Fatalf("got %d orders, want five", len(orders))
	}
	seen := map[string]bool{}
	for _, order := range orders {
		if err := order.validate(); err != nil || seen[order.ID] {
			t.Fatalf("invalid/duplicate fixture: %+v, %v", order, err)
		}
		seen[order.ID] = true
	}
	for _, order := range []Order{{}, {ID: "order-1", FailAt: "unknown"}} {
		called := false
		if _, err := processOrder(order, func(string, Order) (bool, error) {
			called = true
			return true, nil
		}); err == nil || called {
			t.Fatalf("invalid order reached an activity: %+v", order)
		}
	}

	for _, activity := range []task.Activity{checkInventory, chargePayment, shipOrder, notifyCustomer} {
		if _, err := activity(activityInput(`{`)); err == nil {
			t.Fatal("malformed activity input accepted")
		}
	}
}

func TestDemoOrders(t *testing.T) {
	output, err := getOrders(nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []Order{{ID: "order-1"}, {ID: "order-2"}}
	if !reflect.DeepEqual(output, want) {
		t.Fatalf("demo orders = %+v, want %+v", output, want)
	}
}
