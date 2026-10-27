package main

import (
	"errors"

	"github.com/microsoft/durabletask-go/task"
)

const squareName = "GoHistoryExportSquare"

func square(ctx task.ActivityContext) (any, error) {
	var n int
	if err := ctx.GetInput(&n); err != nil {
		return nil, err
	}
	if n < 1 || n > sourceCount {
		return nil, errors.New("this sample accepts only inputs 1 through 5")
	}
	return n * n, nil
}
