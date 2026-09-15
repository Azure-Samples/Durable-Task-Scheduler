package main

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/microsoft/durabletask-go/api"
)

func TestVersionBranches(t *testing.T) {
	for _, test := range []struct {
		version string
		want    []string
	}{
		{"1.0.0", []string{helloName}},
		{"2.0.0", []string{helloName, goodbyeName}},
		{"3.0.0", []string{helloName, goodbyeName, notificationName}},
		{"10.0.0", []string{helloName, goodbyeName, notificationName}},
	} {
		got, err := stepsForVersion(test.version)
		if err != nil || !reflect.DeepEqual(got, test.want) {
			t.Fatalf("%s: steps = %v, %v", test.version, got, err)
		}
	}
	for _, unsupported := range []string{"", "3.0.0-preview", "11.0.0", "garbage"} {
		if _, err := stepsForVersion(unsupported); err == nil {
			t.Fatalf("unsupported version %q succeeded", unsupported)
		}
	}
}

func TestVersionedRegistrations(t *testing.T) {
	registry, err := newRegistry()
	if err != nil {
		t.Fatal(err)
	}
	snapshot := registry.Snapshot()
	if len(snapshot.Orchestrators) != len(versions) || len(snapshot.Activities) != 3*len(versions) || len(snapshot.Entities) != 0 {
		t.Fatalf("unexpected registry: %+v", snapshot)
	}
	for _, registration := range append(snapshot.Orchestrators, snapshot.Activities...) {
		if _, err := stepsForVersion(registration.Version); err != nil {
			t.Fatalf("registration %v: %v", registration, err)
		}
	}
}

type activityInput struct {
	ctx  context.Context
	data string
}

func (a activityInput) Context() context.Context { return a.ctx }
func (a activityInput) GetInput(target any) error {
	return json.Unmarshal([]byte(a.data), target)
}

func TestActivityObservesInheritedVersion(t *testing.T) {
	activity := messageActivity("Hello, %s!")
	for _, version := range versions {
		ctx := api.WithActivityContextInfo(context.Background(), api.ActivityContextInfo{Version: version})
		got, err := activity(activityInput{ctx: ctx, data: `"World"`})
		if err != nil || got != (activityResult{Message: "Hello, World!", Version: version}) {
			t.Fatalf("activity result = %+v, %v", got, err)
		}
	}
	if _, err := activity(activityInput{ctx: context.Background(), data: `"World"`}); err == nil {
		t.Fatal("missing version metadata was accepted")
	}
}

func TestRejectWrongVersionAndOutput(t *testing.T) {
	valid := versionResult{Version: "1.0.0", Results: []string{"Hello, World!"}, ActivityVersions: []string{"1.0.0"}}
	if err := validateResult(valid, "1.0.0"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []versionResult{
		{Version: "2.0.0", Results: valid.Results, ActivityVersions: valid.ActivityVersions},
		{Version: "1.0.0", Results: []string{"wrong"}, ActivityVersions: valid.ActivityVersions},
		{Version: "1.0.0", Results: valid.Results},
		{Version: "1.0.0", Results: valid.Results, ActivityVersions: []string{currentVersion}},
	} {
		if err := validateResult(bad, "1.0.0"); err == nil {
			t.Fatalf("invalid result was accepted: %+v", bad)
		}
	}
}
