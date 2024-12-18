package service

import (
	"errors"
	"time"

	"github.com/ozoli99/Kaida/db"
	"github.com/ozoli99/Kaida/models"
)

type AppointmentService struct {
	Database db.Database
}

func (service *AppointmentService) Create(appointment models.Appointment) (int, error) {
	return service.Database.CreateAppointment(appointment)
}

func (service *AppointmentService) GetAll(limit, offset int, filters map[string]interface{}, sort string) ([]models.Appointment, error) {
	return service.Database.GetAllAppointments(limit, offset, filters, sort)
}

func (service *AppointmentService) Update(appointment models.Appointment) error {
	return service.Database.UpdateAppointment(appointment)
}

func (service *AppointmentService) Delete(appointmentID int) error {
	return service.Database.DeleteAppointment(appointmentID)
}

func (service *AppointmentService) CheckForConflict(appointment models.Appointment) error {
	existingAppointments, err := service.Database.GetAppointmentsByCustomerAndTimeRange(appointment.CustomerName, appointment.Time, appointment.Time.Add(time.Duration(appointment.Duration)*time.Minute))
	if err != nil {
		return err
	}
	if len(existingAppointments) > 0 {
		return errors.New("conflict detected: overlapping appointment")
	}
	return nil
}