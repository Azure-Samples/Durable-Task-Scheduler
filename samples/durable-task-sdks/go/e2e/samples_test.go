package e2e

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

var samples = []string{
	"function-chaining",
	"fan-out-fan-in",
	"human-interaction",
	"monitoring",
	"eternal-orchestrations",
	"sub-orchestrations",
	"bounded-coordinator",
	"saga",
	"async-http-api",
	"entities",
	"versioning",
	"work-item-filtering",
	"orchestration-management",
	"scheduled-tasks",
	"large-payload",
	"history-export",
	"opentelemetry-tracing",
	"arXiv_research_agent",
	"testing",
}

func TestSampleCatalog(t *testing.T) {
	entries, err := os.ReadDir("..")
	if err != nil {
		t.Fatal(err)
	}
	var programs []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if _, err := os.Stat(filepath.Join("..", entry.Name(), "main.go")); os.IsNotExist(err) {
			continue
		} else if err != nil {
			t.Fatalf("sample %s has no readable entrypoint: %v", entry.Name(), err)
		}
		programs = append(programs, entry.Name())
	}
	expected := slices.Clone(samples)
	slices.Sort(expected)
	slices.Sort(programs)
	if !slices.Equal(expected, programs) {
		t.Fatalf("update sample catalog and E2E coverage: catalog=%v, programs=%v", expected, programs)
	}
	for _, name := range samples {
		for _, file := range []string{"main.go", "integration_test.go", "README.md"} {
			if _, err := os.Stat(filepath.Join("..", name, file)); err != nil {
				t.Errorf("%s/%s: %v", name, file, err)
			}
		}
	}
}

func TestSamples(t *testing.T) {
	if os.Getenv("DTS_SAMPLES_E2E") != "1" {
		t.Skip("set DTS_SAMPLES_E2E=1 with a running DTS backend and Azurite")
	}
	for _, name := range samples {
		t.Run(name, func(t *testing.T) {
			// Keep workers sequential: history export must not overlap other
			// workloads in the same hub.
			t.Run("demo", func(t *testing.T) {
				ctx, cancel := context.WithTimeout(t.Context(), 4*time.Minute)
				defer cancel()
				binary := executablePath(t, "sample")
				buildGo(t, ctx, "build", "-mod=readonly", "-o", binary, "./"+name)
				output := runExecutable(t, ctx, name, binary, "-timeout", "3m")
				if strings.TrimSpace(string(output)) == "" {
					t.Fatal("demo did not display a result")
				}
				t.Logf("%s", output)
			})
			t.Run("integration", func(t *testing.T) {
				ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
				defer cancel()
				binary := executablePath(t, "integration")
				buildGo(t, ctx, "test", "-c", "-mod=readonly", "-o", binary, "./"+name)
				output := runExecutable(t, ctx, name, binary,
					"-test.v", "-test.run", "^TestIntegration$", "-test.timeout", "4m")
				if !integrationPassed(string(output)) {
					t.Fatalf("TestIntegration did not run and pass without skips:\n%s", output)
				}
				t.Logf("%s", output)
			})
		})
	}
}

func executablePath(t *testing.T, name string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(t.TempDir(), name)
}

func buildGo(t *testing.T, ctx context.Context, args ...string) {
	t.Helper()
	command := exec.CommandContext(ctx, "go", args...)
	command.Dir = ".."
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
}

func runExecutable(t *testing.T, ctx context.Context, name, binary string, args ...string) []byte {
	t.Helper()
	// Run the binary directly so cancellation cannot orphan a worker beneath
	// a terminated go-run or go-test wrapper.
	command := exec.CommandContext(ctx, binary, args...)
	command.Dir = filepath.Join("..", name)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s failed: %v\n%s", name, err, output)
	}
	return output
}

func integrationPassed(output string) bool {
	ran, passed := false, false
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		if fields[0] == "---" && fields[1] == "SKIP:" {
			return false
		}
		if fields[0] == "===" && fields[1] == "RUN" && fields[2] == "TestIntegration" {
			ran = true
		}
		if fields[0] == "---" && fields[1] == "PASS:" && fields[2] == "TestIntegration" {
			passed = true
		}
	}
	return ran && passed
}

func TestIntegrationResultDetection(t *testing.T) {
	for _, test := range []struct {
		name   string
		output string
		want   bool
	}{
		{"passed", "=== RUN   TestIntegration\n--- PASS: TestIntegration (0.10s)\nPASS\n", true},
		{"no tests", "testing: warning: no tests to run\nPASS\n", false},
		{"skipped", "=== RUN   TestIntegration\n--- SKIP: TestIntegration (0.00s)\nPASS\n", false},
		{"skipped case", "=== RUN   TestIntegration\n--- PASS: TestIntegration (0.10s)\n    --- SKIP: TestIntegration/case (0.00s)\n", false},
		{"different test", "=== RUN   TestIntegrationElsewhere\n--- PASS: TestIntegrationElsewhere (0.10s)\n", false},
		{"failed", "=== RUN   TestIntegration\n--- FAIL: TestIntegration (0.10s)\nFAIL\n", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := integrationPassed(test.output); got != test.want {
				t.Fatalf("integrationPassed = %t, want %t", got, test.want)
			}
		})
	}
}
