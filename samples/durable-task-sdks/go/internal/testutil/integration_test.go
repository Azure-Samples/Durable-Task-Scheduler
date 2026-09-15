package testutil

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestIntegrationContextDisabled(t *testing.T) {
	t.Setenv("DTS_SAMPLES_E2E", "0")
	ran := false
	t.Run("disabled", func(t *testing.T) {
		IntegrationContext(t)
		ran = true
	})
	if ran {
		t.Fatal("integration test ran without explicit opt-in")
	}
}

func TestIntegrationContextLifetime(t *testing.T) {
	t.Setenv("DTS_SAMPLES_E2E", "1")
	var ctx context.Context
	t.Run("enabled", func(t *testing.T) {
		ctx = IntegrationContext(t)
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > 2*time.Minute {
			t.Fatal("integration context must have a bounded deadline")
		}
	})
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatal("integration context was not canceled after the test")
	}
}

func TestRequire(t *testing.T) {
	if err := Require(true, "unexpected error"); err != nil {
		t.Fatal(err)
	}
	if err := Require(false, "got %d", 42); err == nil || err.Error() != "got 42" {
		t.Fatalf("unexpected assertion error: %v", err)
	}
}
