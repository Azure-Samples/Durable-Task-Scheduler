package main

import (
	"errors"
	"strings"

	"github.com/microsoft/durabletask-go/task"
)

const record = "RECORD|"

type payloadResult struct {
	Content string `json:"content"`
	Records int    `json:"records"`
	Bytes   int    `json:"bytes"`
}

func echoData(ctx task.ActivityContext) (any, error) {
	var content string
	if err := ctx.GetInput(&content); err != nil {
		return nil, err
	}
	return content, nil
}

func processData(ctx task.ActivityContext) (any, error) {
	var content string
	if err := ctx.GetInput(&content); err != nil {
		return nil, err
	}
	count := strings.Count(content, record)
	expected, err := recordData(count)
	if err != nil {
		return nil, err
	}
	if expected != content {
		return nil, errors.New("input contains malformed records")
	}
	return payloadResult{Content: content, Records: count, Bytes: len(content)}, nil
}

func recordData(count int) (string, error) {
	// Leave room for the JSON envelope within the SDK's configured payload cap.
	if count < 0 || count > (maxPayloadBytes-1024)/len(record) {
		return "", errors.New("record count exceeds the sample's payload limit")
	}
	return strings.Repeat(record, count), nil
}
