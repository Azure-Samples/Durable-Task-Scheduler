package main

import (
	"errors"

	"github.com/microsoft/durabletask-go/task"
)

const (
	orchestrationName = "GoSagaTravelBooking"
	bookFlightName    = "GoSagaBookFlight"
	bookHotelName     = "GoSagaBookHotel"
	bookCarName       = "GoSagaBookCar"
	cancelFlightName  = "GoSagaCancelFlight"
	cancelHotelName   = "GoSagaCancelHotel"
	cancelCarName     = "GoSagaCancelCar"
)

func newRegistry() (*task.TaskRegistry, error) {
	r := task.NewTaskRegistry()
	return r, errors.Join(
		r.AddOrchestratorN(orchestrationName, travelBookingSaga),
		r.AddActivityN(bookFlightName, bookFlight),
		r.AddActivityN(bookHotelName, bookHotel),
		r.AddActivityN(bookCarName, bookCar),
		r.AddActivityN(cancelFlightName, cancelFlight),
		r.AddActivityN(cancelHotelName, cancelHotel),
		r.AddActivityN(cancelCarName, cancelCar),
	)
}
