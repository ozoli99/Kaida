package db

import "github.com/ozoli99/Kaida/models"

type Database interface {
	Init() error
	CreateAppointment(a models.Appointment) (int, error)
	GetAllAppointments(limit, offset int) ([]models.Appointment, error)
	UpdateAppointment(a models.Appointment) error
	DeleteAppointment(id int) error
}