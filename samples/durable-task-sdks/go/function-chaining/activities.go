package main

import (
	"errors"
	"strings"

	"github.com/microsoft/durabletask-go/task"
)

type Greeting struct {
	Recipient string `json:"recipient"`
	Message   string `json:"message"`
}

func sayHello(ctx task.ActivityContext) (any, error) {
	var name string
	if err := ctx.GetInput(&name); err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("recipient must not be empty")
	}
	return Greeting{Recipient: name, Message: "Hello " + name + "!"}, nil
}

func readGreeting(ctx task.ActivityContext) (Greeting, error) {
	var greeting Greeting
	if err := ctx.GetInput(&greeting); err != nil {
		return Greeting{}, err
	}
	if greeting.Recipient == "" || greeting.Message == "" {
		return Greeting{}, errors.New("greeting requires a recipient and message")
	}
	return greeting, nil
}

func processGreeting(ctx task.ActivityContext) (any, error) {
	greeting, err := readGreeting(ctx)
	if err != nil {
		return nil, err
	}
	greeting.Message += " How are you today?"
	return greeting, nil
}

func finalizeResponse(ctx task.ActivityContext) (any, error) {
	greeting, err := readGreeting(ctx)
	if err != nil {
		return nil, err
	}
	greeting.Message += " I hope you're doing well!"
	return greeting, nil
}
