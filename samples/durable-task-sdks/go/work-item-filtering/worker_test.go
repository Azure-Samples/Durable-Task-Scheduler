package main

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/microsoft/durabletask-go/task"
)

func TestDisjointRegistrations(t *testing.T) {
	a, b, err := newRegistries()
	if err != nil {
		t.Fatal(err)
	}
	wantA := task.TaskRegistrySnapshot{
		Orchestrators: []task.TaskRegistration{{Name: greetingName}},
		Activities:    []task.TaskRegistration{{Name: helloName}},
		Entities:      []string{},
	}
	wantB := task.TaskRegistrySnapshot{
		Orchestrators: []task.TaskRegistration{{Name: mathName}},
		Activities:    []task.TaskRegistration{{Name: addName}},
		Entities:      []string{},
	}
	if !reflect.DeepEqual(a.Snapshot(), wantA) || !reflect.DeepEqual(b.Snapshot(), wantB) {
		t.Fatalf("workers do not have the required disjoint registrations: A=%+v B=%+v", a.Snapshot(), b.Snapshot())
	}
}

type activityInput string

func (a activityInput) Context() context.Context { return context.Background() }
func (a activityInput) GetInput(target any) error {
	return json.Unmarshal([]byte(a), target)
}

func TestWorkerActivities(t *testing.T) {
	greeting, err := sayHello(activityInput(`"World"`))
	if err != nil || greeting != (greetingResult{Worker: "A", Result: "Hello, World!"}) {
		t.Fatalf("greeting = %+v, %v", greeting, err)
	}
	for _, test := range []struct {
		input string
		want  int
	}{{`{"a":40,"b":2}`, 42}, {`{"a":-3,"b":5}`, 2}, {`{"a":0,"b":0}`, 0}} {
		result, err := addNumbers(activityInput(test.input))
		if err != nil || result != (mathResult{Worker: "B", Result: test.want}) {
			t.Fatalf("math = %+v, %v", result, err)
		}
	}
	if _, err := sayHello(activityInput(`{}`)); err == nil {
		t.Fatal("greeting accepted non-string input")
	}
	if _, err := addNumbers(activityInput(`{"a":"forty"}`)); err == nil {
		t.Fatal("math accepted non-numeric input")
	}
}
