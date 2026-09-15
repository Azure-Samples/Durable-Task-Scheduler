package main

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
)

type localSteps struct {
	calls       []string
	paymentErr  error
	shipmentErr error
}

func (s *localSteps) Validate(input order) error {
	s.calls = append(s.calls, "validate")
	return validateOrder(input)
}

func (s *localSteps) Charge(amount int64) (string, error) {
	s.calls = append(s.calls, "charge")
	if s.paymentErr != nil {
		return "", s.paymentErr
	}
	return chargePayment(amount)
}

func (s *localSteps) Ship(input shipment) (string, error) {
	s.calls = append(s.calls, "ship")
	if s.shipmentErr != nil {
		return "", s.shipmentErr
	}
	return shipOrder(input)
}

func TestOrderProcessing(t *testing.T) {
	for _, test := range []struct {
		name  string
		input order
		want  orderResult
	}{
		{"single", order{"Alice", []item{{"Widget", 2, 1000}}}, orderResult{"PAY-2000", "TRACK-ALICE-1", 2000, "completed"}},
		{"multiple", order{"Bob", []item{{"Widget", 3, 2500}, {"Gadget", 1, 9999}}}, orderResult{"PAY-17499", "TRACK-BOB-2", 17499, "completed"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			steps := &localSteps{}
			got, err := processOrder(test.input, steps)
			if err != nil || got != test.want {
				t.Fatalf("processOrder = %+v, %v; want %+v", got, err, test.want)
			}
			if !reflect.DeepEqual(steps.calls, []string{"validate", "charge", "ship"}) {
				t.Fatalf("unexpected activity order: %v", steps.calls)
			}
		})
	}
}

func TestValidationPreventsSideEffects(t *testing.T) {
	for _, test := range []struct {
		name  string
		input order
		cause string
	}{
		{"customer", order{" ", []item{{"Widget", 1, 1000}}}, "customer name"},
		{"items", order{"Alice", nil}, "at least one item"},
		{"quantity", order{"Alice", []item{{"Widget", 0, 1000}}}, "invalid quantity"},
		{"negative-price", order{"Alice", []item{{"Widget", 1, -1}}}, "invalid quantity or price"},
		{"line-overflow", order{"Alice", []item{{"Widget", math.MaxInt64, 2}}}, "line total"},
		{"total-overflow", order{"Alice", []item{{"Widget", 1, math.MaxInt64}, {"Gadget", 1, 1}}}, "order total"},
	} {
		t.Run(test.name, func(t *testing.T) {
			steps := &localSteps{}
			_, err := processOrder(test.input, steps)
			if err == nil || !strings.Contains(err.Error(), test.cause) {
				t.Fatalf("expected %q, got %v", test.cause, err)
			}
			if !reflect.DeepEqual(steps.calls, []string{"validate"}) {
				t.Fatalf("side effects after validation failure: %v", steps.calls)
			}
		})
	}
}

func TestPaymentFailurePreventsShipping(t *testing.T) {
	expected := errors.New("payment declined")
	steps := &localSteps{paymentErr: expected}
	_, err := processOrder(order{"Alice", []item{{"Widget", 1, 1000}}}, steps)
	if !errors.Is(err, expected) || !reflect.DeepEqual(steps.calls, []string{"validate", "charge"}) {
		t.Fatalf("err=%v, calls=%v", err, steps.calls)
	}
}

func TestShipmentFailureIsReturned(t *testing.T) {
	expected := errors.New("shipping unavailable")
	steps := &localSteps{shipmentErr: expected}
	_, err := processOrder(order{"Alice", []item{{"Widget", 1, 1000}}}, steps)
	if !errors.Is(err, expected) {
		t.Fatalf("expected shipment error, got %v", err)
	}
}
