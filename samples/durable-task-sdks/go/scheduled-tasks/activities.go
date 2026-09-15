package main

import (
	"errors"
	"fmt"

	"github.com/microsoft/durabletask-go/task"
)

func sendReport(ctx task.ActivityContext) (any, error) {
	var input reportInput
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	if input.ScheduleID == "" || input.Region == "" ||
		(input.Phase != "initial" && input.Phase != "updated") {
		return nil, errors.New("schedule_id, region, and a recognized phase are required")
	}
	result := reportResult{
		ScheduleID: input.ScheduleID,
		Phase:      input.Phase,
		Message:    fmt.Sprintf("Report for '%s' generated", input.Region),
	}
	fmt.Println(result.Message)
	return result, nil
}
