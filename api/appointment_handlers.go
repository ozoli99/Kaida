package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/ozoli99/Kaida/models"
)

func (server *Server) handleAppointments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
		case http.MethodGet:
			server.getAllAppointments(w, r)
		case http.MethodPost:
			server.createAppointment(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (server *Server) handleAppointmentByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/appointments/"):]
	appointmentID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
		case http.MethodGet:
			server.getAppointmentByID(w, appointmentID)
		case http.MethodPut:
			server.updateAppointment(w, r, appointmentID)
		case http.MethodDelete:
			server.deleteAppointment(w, appointmentID)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (server *Server) handleRecurringAppointments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
		case http.MethodGet:
			limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
			if limit <= 0 {
				limit = 10
			}

			recurring, err := server.AppointmentService.GetFutureOccurrences(limit)
			if err != nil {
				http.Error(w, "Failed to fetch recurring appointments", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(recurring)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (server *Server) getAllAppointments(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit <= 0 {
		limit = 10
	}
	offset, _ := strconv.Atoi(query.Get("offset"))
	if offset < 0 {
		offset = 0
	}

	filters := map[string]interface{}{}
	if customerName := query.Get("customer_name"); customerName != "" {
		filters["customer_name"] = customerName
	}
	if startTime := query.Get("start"); startTime != "" {
		filters["start_time"] = startTime
	}
	if endTime := query.Get("end"); endTime != "" {
		filters["end_time"] = endTime
	}

	sortCriteria := query.Get("sort")

	appointments, err := server.AppointmentService.GetAll(limit, offset, filters, sortCriteria)
	if err != nil {
		http.Error(w, "Failed to fetch appointments", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(appointments)
}

func (server *Server) createAppointment(w http.ResponseWriter, r *http.Request) {
	var newAppointment models.Appointment
	if err := json.NewDecoder(r.Body).Decode(&newAppointment); err != nil {
		log.Printf("Failed to decode input: %v", err)
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	if newAppointment.RecurrenceRule == "" {
		newAppointment.RecurrenceRule = "None"
	}
	if newAppointment.Status == "" {
		newAppointment.Status = "Scheduled"
	}
	if newAppointment.Resource == "" {
		newAppointment.Resource = ""
	}

	if err := newAppointment.Validate(); err != nil {
		log.Printf("Validation error: %v", err)
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
		return
	}
	if err := server.AppointmentService.CheckForConflict(newAppointment); err != nil {
		log.Printf("Conflict error: %v", err)
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusConflict)
		return
	}

	id, err := server.AppointmentService.Create(newAppointment)
	if err != nil {
		http.Error(w, "Failed to create appointment", http.StatusInternalServerError)
		return
	}
	newAppointment.ID = id
	if server.WebSocketServer != nil {
		message, _ := json.Marshal(newAppointment)
		server.WebSocketServer.Broadcast(message)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newAppointment)
}

func (server *Server) getAppointmentByID(w http.ResponseWriter, appointmentID int) {
	filters := map[string]interface{}{
		"id": appointmentID,
	}
	appointments, err := server.AppointmentService.GetAll(1, 0, filters, "")
	if err != nil || len(appointments) == 0 {
		http.Error(w, "Appointment not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(appointments[0])
}

func (server *Server) updateAppointment(w http.ResponseWriter, r *http.Request, appointmentID int) {
	var updatedAppointment models.Appointment
	if err := json.NewDecoder(r.Body).Decode(&updatedAppointment); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	if err := updatedAppointment.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	updatedAppointment.ID = appointmentID
	if err := server.AppointmentService.Update(updatedAppointment); err != nil {
		http.Error(w, "Failed to update appointment", http.StatusInternalServerError)
		return
	}
	if server.WebSocketServer != nil {
		message, _ := json.Marshal(updatedAppointment)
		server.WebSocketServer.Broadcast(message)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedAppointment)
}

func (server *Server) updateAppointmentStatus(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/appointments/"):]
	appointmentID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid appointment ID", http.StatusBadRequest)
		return
	}

	var statusUpdate struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&statusUpdate); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	if err := server.AppointmentService.UpdateAppointmentStatus(appointmentID, statusUpdate.Status); err != nil {
		http.Error(w, "Failed to update appointment status", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (server *Server) deleteAppointment(w http.ResponseWriter, appointmentID int) {
	if err := server.AppointmentService.Delete(appointmentID); err != nil {
		http.Error(w, "Failed to delete appointment", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}