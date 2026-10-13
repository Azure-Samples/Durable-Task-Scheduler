package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type activityInput []byte

func (input activityInput) GetInput(target any) error { return json.Unmarshal(input, target) }
func (activityInput) Context() context.Context        { return context.Background() }

func TestJobStatusSequence(t *testing.T) {
	for _, completesAfter := range []int{0, 4} {
		for count := range 5 {
			data, err := json.Marshal(CheckInput{JobID: "job-1", CheckCount: count, CompleteAfterChecks: completesAfter})
			if err != nil {
				t.Fatal(err)
			}
			output, err := checkJobStatus(activityInput(data))
			if err != nil {
				t.Fatal(err)
			}
			got := output.(JobStatus)
			want := "Running"
			if completesAfter != 0 && count+1 >= completesAfter {
				want = "Completed"
			}
			if got.JobID != "job-1" || got.CheckCount != count+1 || got.Status != want {
				t.Fatalf("check %d (complete after %d) = %+v", count, completesAfter, got)
			}
		}
	}
}

func TestPollDeadlineClamping(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, scenario := range []struct {
		untilDeadline time.Duration
		want          time.Duration
	}{
		{5 * time.Second, 2 * time.Second},
		{time.Second, time.Second},
		{0, 0},
		{-time.Second, 0},
	} {
		if got := nextPollDelay(now, now.Add(scenario.untilDeadline), 2*time.Second); got != scenario.want {
			t.Fatalf("delay with %v remaining = %v, want %v", scenario.untilDeadline, got, scenario.want)
		}
	}
}

func TestInvalidMonitoringInputs(t *testing.T) {
	valid := MonitorRequest{JobID: "job-1", PollIntervalMilliseconds: 250, TimeoutMilliseconds: 1000, CompleteAfterChecks: 4}
	if err := valid.validate(); err != nil {
		t.Fatal(err)
	}
	for _, modify := range []func(*MonitorRequest){
		func(r *MonitorRequest) { r.JobID = "" },
		func(r *MonitorRequest) { r.PollIntervalMilliseconds = 0 },
		func(r *MonitorRequest) { r.TimeoutMilliseconds = -1 },
		func(r *MonitorRequest) { r.TimeoutMilliseconds = int64(25 * time.Hour / time.Millisecond) },
		func(r *MonitorRequest) { r.CompleteAfterChecks = -1 },
		func(r *MonitorRequest) { r.CompleteAfterChecks = 101 },
	} {
		invalid := valid
		modify(&invalid)
		if err := invalid.validate(); err == nil {
			t.Fatalf("invalid monitor accepted: %+v", invalid)
		}
	}
	for _, raw := range []string{`{`, `{"job_id":"job-1","check_count":-1}`, `{}`} {
		if _, err := checkJobStatus(activityInput(raw)); err == nil {
			t.Fatalf("invalid activity payload accepted: %s", raw)
		}
	}
}
