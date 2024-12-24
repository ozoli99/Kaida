package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/ozoli99/Kaida/db"
	"github.com/ozoli99/Kaida/models"
)

type DefaultAppointmentService struct {
	Database db.Database
}

var _ AppointmentService = (*DefaultAppointmentService)(nil)

func (service *DefaultAppointmentService) GetAllAppointments(user *models.User, limit, offset int, filters map[string]interface{}, sort string) ([]models.Appointment, error) {
	switch user.Role {
		case "admin":
		case "customer":
			filters["customer_id"] = user.ID
		case "provider":
			filters["provider_id"]= user.ID
	}

	return service.Database.GetAllAppointments(limit, offset, filters, sort)
}

func (service *DefaultAppointmentService) CheckForConflict(appointment models.Appointment) error {
	existingAppointments, err := service.Database.GetAppointmentsByCustomerAndTimeRange(appointment.CustomerName, appointment.Time, appointment.Time.Add(time.Duration(appointment.Duration)*time.Minute))
	if err != nil {
		return err
	}
	if len(existingAppointments) > 0 {
		return errors.New("conflict detected: overlapping appointment")
	}
	return nil
}

func (service *DefaultAppointmentService) CreateAppointment(user *models.User, appointment models.Appointment) (int, error) {
	if err := service.authorizeCreate(user, appointment); err != nil {
		return 0, err
	}

	insertedID, err := service.Database.CreateAppointment(appointment)
	if err != nil {
		return 0, err
	}

	return insertedID, nil
}

func (service *DefaultAppointmentService) UpdateAppointment(user *models.User, appointment models.Appointment) error {
	existingAppointment, err := service.Database.GetAppointmentByID(appointment.ID)
	if err != nil {
		return err
	}

	if err := service.authorizeUpdate(user, existingAppointment, appointment); err != nil {
		return err
	}

	return service.Database.UpdateAppointment(appointment)
}

func (service *DefaultAppointmentService) UpdateAppointmentStatus(appointmentID int, status string) error {
	return service.Database.UpdateAppointmentStatus(appointmentID, status)
}

func (service *DefaultAppointmentService) DeleteAppointment(user *models.User, appointmentID int) error {
	appointment, err := service.Database.GetAppointmentByID(appointmentID)
	if err != nil {
		return err
	}

	if !service.authorizeDelete(user, appointment) {
		return fmt.Errorf("unauthorized to delete appointment")
	}

	return service.Database.DeleteAppointment(appointmentID)
}

func (service *DefaultAppointmentService) GetFutureOccurrences(limit int) ([]models.Appointment, error) {
	recurringAppointments, err := service.Database.GetRecurringAppointments(limit)
	if err != nil {
		return nil, err
	}

	var futureOccurrences []models.Appointment
	for _, appointment := range recurringAppointments {
		occurrences := appointment.CalculateFutureOccurences(limit)
		for _, occ := range occurrences {
			futureOccurrences = append(futureOccurrences, models.Appointment{
				CustomerName: appointment.CustomerName,
				Time: occ,
				Duration: appointment.Duration,
				Notes: appointment.Notes,
				RecurrenceRule: appointment.RecurrenceRule,
			})
		}
	}
	return futureOccurrences, nil
}

func (service *DefaultAppointmentService) authorizeCreate(user *models.User, appointment models.Appointment) error {
	switch user.Role {
		case "admin":
			return nil
		case "customer":
			if appointment.CustomerID != user.ID {
				return fmt.Errorf("unauthorized: customers can only create appointments for themselves")
			}
			return nil
		case "provider":
			if appointment.ProviderID != user.ID {
				return fmt.Errorf("unauthorized: masseur can only create for themselves as a provider")
			}
			return nil
		default:
			return fmt.Errorf("unauthorized: unknown role %q", user.Role)
	}
}

func (service *DefaultAppointmentService) GetAppointmentByID(appointmentID int) (models.Appointment, error) {
	return service.Database.GetAppointmentByID(appointmentID)
}

func (service *DefaultAppointmentService) authorizeUpdate(user *models.User, oldAppointment, newAppointment models.Appointment) error {
	if user.Role == "admin" {
		return nil
	}

	if user.Role == "customer" {
		if oldAppointment.CustomerID != user.ID {
			return fmt.Errorf("unauthorized: cannot update appointment not owned by you")
		}
		return nil
	}

	if user.Role == "provider" {
		if oldAppointment.ProviderID != user.ID {
			return fmt.Errorf("unauthorized: not assigned as provider for this assignment")
		}
		return nil
	}

	return fmt.Errorf("unauthorized")
}

func (service *DefaultAppointmentService) authorizeDelete(user *models.User, appointment models.Appointment) bool {
	if user.Role == "admin" {
		return true
	}

	if user.Role == "customer" && appointment.CustomerID == user.ID {
		return true
	}

	if user.Role == "provider" && appointment.ProviderID == user.ID {
		return true
	}

	return false
}

func (service *DefaultAppointmentService) MarkAppointmentComplete(user *models.User, appointmentID int) error {
	appointment, err := service.Database.GetAppointmentByID(appointmentID)
	if err != nil {
		return err
	}

	if !(user.Role == "admin" || (user.Role == "provider" && appointment.ProviderID == user.ID)) {
		return fmt.Errorf("unauthorized: you are not the provider or an admin")
	}

	appointment.Status = "Completed"
	return service.Database.UpdateAppointment(appointment)
}