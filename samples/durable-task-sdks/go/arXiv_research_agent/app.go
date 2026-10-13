package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
)

func defaultMode() string {
	if value := strings.TrimSpace(os.Getenv("RESEARCH_MODE")); value != "" {
		return value
	}
	return "fixture"
}

func run(ctx context.Context) error {
	if *serve {
		if _, err := loopbackAddress(*listen); err != nil {
			return err
		}
	}
	if !*serve && *mode == "real" {
		return errors.New("use -serve -mode real for external research; the default demo uses fixtures")
	}
	activities, err := newActivities(*mode)
	if err != nil {
		return err
	}
	registry, err := newRegistry(activities)
	if err != nil {
		return err
	}
	fmt.Printf("Research mode: %s (fixture papers and reports are synthetic, not academic evidence)\n", *mode)
	return sample.WithHost(ctx, registry, func(ctx context.Context, client *dts.Client) error {
		app := &researchAPI{store: schedulerStore{client}, mode: *mode}
		if *serve {
			return serveHTTP(ctx, *listen, app.handler())
		}
		return demo(ctx, app.handler())
	})
}

func newActivities(mode string) (*activities, error) {
	if mode != "fixture" && mode != "real" {
		return nil, errors.New("mode must be fixture or real")
	}
	activities := &activities{mode: mode}
	if mode == "real" {
		config, err := loadModelConfig(os.Getenv)
		if err != nil {
			return nil, err
		}
		model, err := newOpenAIModel(config)
		if err != nil {
			return nil, err
		}
		source, err := newArxivClient(strings.TrimSpace(os.Getenv("ARXIV_API_ENDPOINT")))
		if err != nil {
			return nil, err
		}
		activities.model, activities.source = model, source
	}
	return activities, nil
}
