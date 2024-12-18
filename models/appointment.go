package models

import (
	"errors"
	"time"
)

type Appointment struct {
	ID           int       `json:"id"`
	CustomerName string    `json:"customer_name"`
	Time         time.Time `json:"time"`
	Duration     int       `json:"duration"`
	Notes        string    `json:"notes"`
}

func (appointment *Appointment) Validate() error {
	if appointment.CustomerName == "" {
		return errors.New("customer name cannot be empty")
	}
	if appointment.Time.IsZero() {
		return errors.New("time cannot be empty")
	}
	if appointment.Duration <= 0 {
		return errors.New("duration must be greater than 0")
	}
	return nil
}