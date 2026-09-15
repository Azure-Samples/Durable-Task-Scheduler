// Package testutil provides helpers for opt-in DTS integration tests.
package testutil

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func IntegrationContext(t *testing.T) context.Context {
	t.Helper()
	if os.Getenv("DTS_SAMPLES_E2E") != "1" {
		t.Skip("set DTS_SAMPLES_E2E=1 to run against a configured DTS backend")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	return ctx
}

func Require(condition bool, format string, args ...any) error {
	if !condition {
		return fmt.Errorf(format, args...)
	}
	return nil
}
