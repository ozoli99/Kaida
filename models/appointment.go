package models

import "time"

type Appointment struct {
	ID int `json:"id"`
	CustomerName string `json:"customer_name"`
	Time time.Time `json:"time"`
	Duration int `json:"duration"`
	Notes string `json:"notes"`
}