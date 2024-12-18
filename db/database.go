package db

import (
	"time"

	"github.com/ozoli99/Kaida/models"
)

type Database interface {
	Init() error
	CreateAppointment(a models.Appointment) (int, error)
	GetAllAppointments(limit, offset int, filters map[string]interface{}, sort string) ([]models.Appointment, error)
	GetAppointmentsByCustomerAndTimeRange(customerName string, start, end time.Time) ([]models.Appointment, error)
	UpdateAppointment(a models.Appointment) error
	DeleteAppointment(id int) error
}