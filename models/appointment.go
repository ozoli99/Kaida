package models

import (
	"errors"
	"time"
)

type Appointment struct {
	ID             int       `json:"id"`
	CustomerName   string    `json:"customer_name"`
	Time           time.Time `json:"time"`
	Duration       int       `json:"duration"`
	Notes          string    `json:"notes"`
	RecurrenceRule string    `json:"recurrence_rules"`
	Status         string    `json:"status"`
	Resource       string    `json:"resource"`

	CustomerID     int       `json:"customer_id"`
	ProviderID     int       `json:"provider_id"`
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

func (appointment *Appointment) CalculateFutureOccurences(limit int) []time.Time {
	if appointment.RecurrenceRule == "" {
		return nil
	}

	var occurrences []time.Time
	current := appointment.Time
	for i := 0; i < limit; i++ {
		switch appointment.RecurrenceRule {
			case "daily":
				current = current.AddDate(0, 0, 1)
			case "weekly":
				current = current.AddDate(0, 0, 7)
			case "monthly":
				current = current.AddDate(0, 1, 0)
			default:
				break
		}
		occurrences = append(occurrences, current)
	}
	return occurrences
}