package service

import (
	"errors"
	"time"

	"github.com/ozoli99/Kaida/db"
	"github.com/ozoli99/Kaida/models"
)

type AppointmentService struct {
	DB db.Database
}

func (s *AppointmentService) Create(a models.Appointment) (int, error) {
	return s.DB.CreateAppointment(a)
}

func (s *AppointmentService) GetAll(limit, offset int, filters map[string]interface{}, sort string) ([]models.Appointment, error) {
	return s.DB.GetAllAppointments(limit, offset, filters, sort)
}

func (s *AppointmentService) Update(a models.Appointment) error {
	return s.DB.UpdateAppointment(a)
}

func (s *AppointmentService) Delete(id int) error {
	return s.DB.DeleteAppointment(id)
}

func (s *AppointmentService) CheckForConflict(a models.Appointment) error {
	existingAppointments, err := s.DB.GetAppointmentsByCustomerAndTimeRange(a.CustomerName, a.Time, a.Time.Add(time.Duration(a.Duration)*time.Minute))
	if err != nil {
		return err
	}
	if len(existingAppointments) > 0 {
		return errors.New("conflict detected: overlapping appointment")
	}
	return nil
}