package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/microsoft/durabletask-go/task"
)

type activityInput []byte

func (input activityInput) GetInput(target any) error { return json.Unmarshal(input, target) }
func (activityInput) Context() context.Context        { return context.Background() }

func inputFor(t *testing.T, value any) activityInput {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestTypedGreetingPipeline(t *testing.T) {
	for _, name := range []string{"User", "Ada", "世界"} {
		t.Run(name, func(t *testing.T) {
			var input any = name
			for _, activity := range []task.Activity{sayHello, processGreeting, finalizeResponse} {
				output, err := activity(inputFor(t, input))
				if err != nil {
					t.Fatal(err)
				}
				greeting, ok := output.(Greeting)
				if !ok || greeting.Recipient != name {
					t.Fatalf("typed payload lost its recipient: %#v", output)
				}
				input = greeting
			}
			want := "Hello " + name + "! How are you today? I hope you're doing well!"
			if got := input.(Greeting).Message; got != want {
				t.Fatalf("message = %q, want %q", got, want)
			}
		})
	}
}

func TestGreetingRejectsInvalidPayloads(t *testing.T) {
	for _, activity := range []task.Activity{sayHello, processGreeting, finalizeResponse} {
		if _, err := activity(activityInput(`{`)); err == nil {
			t.Fatal("malformed JSON was accepted")
		}
	}
	if _, err := sayHello(inputFor(t, "  ")); err == nil {
		t.Fatal("empty recipient was accepted")
	}
	for _, activity := range []task.Activity{processGreeting, finalizeResponse} {
		if _, err := activity(inputFor(t, Greeting{Recipient: "User"})); err == nil {
			t.Fatal("missing message was accepted")
		}
	}
}

func TestRegistrationNames(t *testing.T) {
	r, err := newRegistry()
	if err != nil {
		t.Fatal(err)
	}
	snapshot := r.Snapshot()
	if len(snapshot.Orchestrators) != 1 || len(snapshot.Activities) != 3 {
		t.Fatalf("unexpected registrations: %+v", snapshot)
	}
	for _, entry := range append(snapshot.Orchestrators, snapshot.Activities...) {
		if !strings.HasPrefix(entry.Name, "GoFunctionChaining") {
			t.Fatalf("unscoped task name: %s", entry.Name)
		}
	}
}
