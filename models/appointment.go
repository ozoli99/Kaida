package models

import (
	"errors"
	"time"
)

type Appointment struct {
	ID int `json:"id"`
	CustomerName string `json:"customer_name"`
	Time time.Time `json:"time"`
	Duration int `json:"duration"`
	Notes string `json:"notes"`
}

func (a *Appointment) Validate() error {
	if a.CustomerName == "" {
		return errors.New("customer name cannot be empty")
	}
	if a.Time.IsZero() {
		return errors.New("time cannot be empty")
	}
	if a.Duration <= 0 {
		return errors.New("duration must be greater than 0")
	}
	return nil
}