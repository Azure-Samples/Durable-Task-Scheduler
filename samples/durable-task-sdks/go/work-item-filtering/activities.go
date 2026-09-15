package main

import (
	"fmt"

	"github.com/microsoft/durabletask-go/task"
)

func sayHello(ctx task.ActivityContext) (any, error) {
	var name string
	if err := ctx.GetInput(&name); err != nil {
		return nil, err
	}
	return greetingResult{Worker: "A", Result: fmt.Sprintf("Hello, %s!", name)}, nil
}

func addNumbers(ctx task.ActivityContext) (any, error) {
	var input numbers
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	return mathResult{Worker: "B", Result: input.A + input.B}, nil
}
