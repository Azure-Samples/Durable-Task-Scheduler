package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/microsoft/durabletask-go/exporthistory"
)

type fakeCleanupJob struct {
	delete   func(context.Context) error
	describe func(context.Context) (*exporthistory.ExportJobDescription, error)
}

func (fakeCleanupJob) ID() string { return "owned-test-job" }
func (j fakeCleanupJob) Delete(ctx context.Context) error {
	return j.delete(ctx)
}
func (j fakeCleanupJob) Describe(ctx context.Context) (*exporthistory.ExportJobDescription, error) {
	return j.describe(ctx)
}

func TestWorkerLifetimeSurvivesScenarioCancellationThroughCleanup(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		name := "cancel"
		if deadline {
			name = "deadline"
		}
		t.Run(name, func(t *testing.T) {
			type contextKey struct{}
			parent := context.WithValue(t.Context(), contextKey{}, "retained")
			ctx, cancel := context.WithCancel(parent)
			want := context.Canceled
			if deadline {
				cancel()
				ctx, cancel = context.WithTimeout(parent, 10*time.Millisecond)
				want = context.DeadlineExceeded
			}
			defer cancel()
			lifetime := newExportWorkerLifetime(ctx)
			defer lifetime.cancel()
			if _, bounded := lifetime.context.Deadline(); bounded {
				t.Fatal("worker inherited the scenario deadline")
			}
			var order []string
			checkCleanup := func(cleanupCtx context.Context) {
				t.Helper()
				if !errors.Is(ctx.Err(), want) || lifetime.context.Err() != nil || cleanupCtx.Err() != nil {
					t.Fatalf("scenario=%v worker=%v cleanup=%v", ctx.Err(), lifetime.context.Err(), cleanupCtx.Err())
				}
				if limit, ok := cleanupCtx.Deadline(); !ok || time.Until(limit) > cleanupTimeout {
					t.Fatal("cleanup does not have its own bounded deadline")
				}
				if cleanupCtx.Value(contextKey{}) != "retained" {
					t.Fatal("worker/cleanup context lost scenario values")
				}
			}
			job := fakeCleanupJob{
				delete: func(cleanupCtx context.Context) error {
					checkCleanup(cleanupCtx)
					order = append(order, "delete")
					return nil
				},
				describe: func(cleanupCtx context.Context) (*exporthistory.ExportJobDescription, error) {
					checkCleanup(cleanupCtx)
					order = append(order, "verify")
					return nil, exporthistory.ErrJobNotFound
				},
			}
			closeErr := errors.New("host close failed")
			err := func() (err error) {
				defer func() {
					err = errors.Join(err, lifetime.close(func() error {
						if lifetime.context.Err() != nil {
							t.Fatal("worker was canceled before Host.Close")
						}
						order = append(order, "close")
						return closeErr
					}))
				}()
				return withJobCleanup(ctx, lifetime.context, job, func() error {
					if deadline {
						<-ctx.Done()
					} else {
						cancel()
					}
					order = append(order, "work")
					return ctx.Err()
				})
			}()
			if !errors.Is(err, want) || !errors.Is(err, closeErr) {
				t.Fatalf("scenario or shutdown error was lost: %v", err)
			}
			if !errors.Is(lifetime.context.Err(), context.Canceled) {
				t.Fatal("worker was not canceled after Host.Close")
			}
			if !reflect.DeepEqual(order, []string{"work", "delete", "verify", "close"}) {
				t.Fatalf("wrong cleanup order: %v", order)
			}
		})
	}
}

func TestCleanupPreservesWorkDeleteVerificationAndShutdownFailures(t *testing.T) {
	workErr := errors.New("work failed")
	deleteErr := errors.New("delete failed")
	verifyErr := errors.New("verification failed")
	closeErr := errors.New("shutdown failed")
	lifetime := newExportWorkerLifetime(t.Context())
	defer lifetime.cancel()
	var verified bool
	job := fakeCleanupJob{
		delete: func(context.Context) error { return deleteErr },
		describe: func(context.Context) (*exporthistory.ExportJobDescription, error) {
			verified = true
			return nil, verifyErr
		},
	}
	err := func() (err error) {
		defer func() { err = errors.Join(err, lifetime.close(func() error { return closeErr })) }()
		return withJobCleanup(t.Context(), lifetime.context, job, func() error { return workErr })
	}()
	for _, want := range []error{workErr, deleteErr, verifyErr, closeErr} {
		if !errors.Is(err, want) {
			t.Fatalf("lost %v from %v", want, err)
		}
	}
	if !verified {
		t.Fatal("a Delete failure skipped absence verification")
	}
}

func TestAbsenceDoesNotHideDeleteFailure(t *testing.T) {
	deleteErr := errors.New("generation purge failed after entity deletion")
	job := fakeCleanupJob{
		delete: func(context.Context) error { return deleteErr },
		describe: func(context.Context) (*exporthistory.ExportJobDescription, error) {
			return nil, exporthistory.ErrJobNotFound
		},
	}
	if err := deleteAndVerifyJob(t.Context(), job); !errors.Is(err, deleteErr) {
		t.Fatalf("an absent entity hid a Delete failure: %v", err)
	}
}

func TestAbsenceVerificationHonorsCleanupDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Millisecond)
	defer cancel()
	job := fakeCleanupJob{
		delete: func(context.Context) error { return nil },
		describe: func(context.Context) (*exporthistory.ExportJobDescription, error) {
			return &exporthistory.ExportJobDescription{Status: exporthistory.ExportJobStatusActive}, nil
		},
	}
	if err := deleteAndVerifyJob(ctx, job); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("absence verification did not honor the deadline: %v", err)
	}
}

func TestCancellationDuringCleanupCannotReportSuccess(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	lifetime := newExportWorkerLifetime(ctx)
	defer lifetime.cancel()
	job := fakeCleanupJob{
		delete: func(cleanupCtx context.Context) error {
			cancel()
			return cleanupCtx.Err()
		},
		describe: func(cleanupCtx context.Context) (*exporthistory.ExportJobDescription, error) {
			if cleanupCtx.Err() != nil {
				t.Fatal("cleanup inherited a late scenario cancellation")
			}
			return nil, exporthistory.ErrJobNotFound
		},
	}
	if err := withJobCleanup(ctx, lifetime.context, job, func() error { return nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("late cancellation reported success: %v", err)
	}
}

func TestPausedWriteSignalsOnceAndReleasesOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	lifetime := newExportWorkerLifetime(ctx)
	defer lifetime.cancel()
	active := make(chan struct{})
	gate := pauseBeforeWrite(ctx, func() { close(active) })
	finished := make(chan error, 2)
	for range 2 {
		go func() { finished <- gate(lifetime.context) }()
	}
	select {
	case <-active:
	case <-time.After(time.Second):
		t.Fatal("paused export did not emit its active-stage signal")
	}
	select {
	case err := <-finished:
		t.Fatalf("write did not remain paused until cancellation: %v", err)
	default:
	}
	cancel()
	for range 2 {
		select {
		case err := <-finished:
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("unexpected gate result: %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("write remained blocked after scenario cancellation")
		}
	}
	if lifetime.context.Err() != nil {
		t.Fatal("releasing paused writes canceled the worker needed by cleanup")
	}
}
