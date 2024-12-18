package db

import (
	"time"

	"github.com/ozoli99/Kaida/models"
)

type Database interface {
	InitializeDatabase() error
	CreateAppointment(appointment models.Appointment) (int, error)
	GetAllAppointments(limit, offset int, filters map[string]interface{}, sort string) ([]models.Appointment, error)
	GetAppointmentsByCustomerAndTimeRange(customerName string, startTime, endTime time.Time) ([]models.Appointment, error)
	GetRecurringAppointments(limit int) ([]models.Appointment, error)
	UpdateAppointment(appointment models.Appointment) error
	DeleteAppointment(appointmentID int) error
}