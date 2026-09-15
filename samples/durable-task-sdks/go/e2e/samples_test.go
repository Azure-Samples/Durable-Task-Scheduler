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
	"agent-directed-workflows",
	"arXiv_research_agent",
	"testing",
}

func TestPythonSampleParity(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join("..", "..", "python"))
	if err != nil {
		t.Fatal(err)
	}
	var pythonSamples []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if _, err := os.Stat(filepath.Join("..", "..", "python", entry.Name(), "README.md")); os.IsNotExist(err) {
			continue
		} else if err != nil {
			t.Fatalf("Python sample %s has no readable README: %v", entry.Name(), err)
		}
		pythonSamples = append(pythonSamples, entry.Name())
	}
	expected := slices.Clone(samples)
	slices.Sort(expected)
	slices.Sort(pythonSamples)
	if !slices.Equal(expected, pythonSamples) {
		t.Fatalf("update Go counterparts and E2E coverage: Go=%v, Python=%v", expected, pythonSamples)
	}
	for _, name := range samples {
		for _, file := range []string{"main.go", "README.md"} {
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
			// Each executable owns its worker and assertions; run sequentially so
			// system workers from one sample cannot consume another's work.
			ctx, cancel := context.WithTimeout(t.Context(), 4*time.Minute)
			defer cancel()
			binary := filepath.Join(t.TempDir(), "sample")
			if runtime.GOOS == "windows" {
				binary += ".exe"
			}
			build := exec.CommandContext(ctx, "go", "build", "-mod=readonly", "-o", binary, "./"+name)
			build.Dir = ".."
			if output, err := build.CombinedOutput(); err != nil {
				t.Fatalf("build sample: %v\n%s", err, output)
			}
			// Execute the binary directly so cancellation cannot orphan a
			// worker beneath a terminated "go run" subprocess.
			command := exec.CommandContext(ctx, binary, "-timeout", "3m")
			command.Dir = filepath.Join("..", name)
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("sample failed: %v\n%s", err, output)
			}
			marker := "SAMPLE_OK " + name
			if !slices.Contains(strings.Split(strings.TrimSpace(string(output)), "\n"), marker) {
				t.Fatalf("sample exited without verification marker %q:\n%s", marker, output)
			}
			t.Logf("%s", output)
		})
	}
}
