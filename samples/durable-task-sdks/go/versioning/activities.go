package main

import (
	"errors"
	"fmt"

	"github.com/microsoft/durabletask-go/api"
	"github.com/microsoft/durabletask-go/task"
)

type activityResult struct {
	Message string `json:"message"`
	Version string `json:"version"`
}

func messageActivity(format string) task.Activity {
	return func(ctx task.ActivityContext) (any, error) {
		var name string
		if err := ctx.GetInput(&name); err != nil {
			return nil, err
		}
		info, ok := api.ActivityContextInfoFromContext(ctx.Context())
		if !ok {
			return nil, errors.New("activity version metadata is missing")
		}
		return activityResult{Message: fmt.Sprintf(format, name), Version: info.Version}, nil
	}
}
