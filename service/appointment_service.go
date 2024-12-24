package service

import "github.com/ozoli99/Kaida/models"

type AppointmentReader interface {
	GetAllAppointments(currentUser *models.User, limit, offset int, filters map[string]interface{}, sort string) ([]models.Appointment, error)
	GetAppointmentByID(appointmentID int) (models.Appointment, error)
}

type AppointmentWriter interface {
	CreateAppointment(currentUser *models.User, appointment models.Appointment) (int, error)
	UpdateAppointment(currentUser *models.User, appointment models.Appointment) error
	UpdateAppointmentStatus(appointmentID int, status string) error
	DeleteAppointment(currentUser *models.User, appointmentID int) error
}

type AppointmentService interface {
	AppointmentReader
	AppointmentWriter
	
	CheckForConflict(appointment models.Appointment) error
	GetFutureOccurrences(limit int) ([]models.Appointment, error)
}