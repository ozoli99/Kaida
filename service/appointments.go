package service

import (
	"github.com/ozoli99/Kaida/db"
	"github.com/ozoli99/Kaida/models"
)

type AppointmentService struct {
	DB db.Database
}

func (s *AppointmentService) Create(a models.Appointment) (int, error) {
	return s.DB.CreateAppointment(a)
}

func (s *AppointmentService) GetAll(limit, offset int) ([]models.Appointment, error) {
	return s.DB.GetAllAppointments(limit, offset)
}

func (s *AppointmentService) Update(a models.Appointment) error {
	return s.DB.UpdateAppointment(a)
}

func (s *AppointmentService) Delete(id int) error {
	return s.DB.DeleteAppointment(id)
}