package api

import (
	"encoding/json"
	"fmt"
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
			writeJSONError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (server *Server) handleAppointmentByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/appointments/"):]
	appointmentID, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSONError(w, "Invalid appointment ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
		case http.MethodGet:
			server.getAppointmentByID(w, r, appointmentID)
		case http.MethodPut:
			server.updateAppointment(w, r, appointmentID)
		case http.MethodDelete:
			server.deleteAppointment(w, r, appointmentID)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (server *Server) handleRecurringAppointments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	recurring, err := server.AppointmentService.GetFutureOccurrences(limit)
	if err != nil {
		writeJSONError(w, "Failed to fetch recurring appointments", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recurring)
}

func (server *Server) getAllAppointments(w http.ResponseWriter, r *http.Request) {
	currentUser, err := server.getCurrentUser(r)
    if err != nil {
        writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

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

	appointments, err := server.AppointmentService.GetAllAppointments(
		currentUser,
		limit,
		offset,
		filters,
		sortCriteria,
	)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(appointments)
}

func (server *Server) createAppointment(w http.ResponseWriter, r *http.Request) {
	currentUser, err := server.getCurrentUser(r)
	if err != nil {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var newAppointment models.Appointment
	if err := json.NewDecoder(r.Body).Decode(&newAppointment); err != nil {
		writeJSONError(w, fmt.Sprintf("Invalid input: %v", err), http.StatusBadRequest)
		return
	}

	switch currentUser.Role {
		case "customer":
			newAppointment.CustomerID = currentUser.ID
		case "provider":
			newAppointment.ProviderID = currentUser.ID
		case "admin":
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
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := server.AppointmentService.CheckForConflict(newAppointment); err != nil {
		writeJSONError(w, err.Error(), http.StatusConflict)
		return
	}

	id, err := server.AppointmentService.CreateAppointment(currentUser, newAppointment)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
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

func (server *Server) getAppointmentByID(w http.ResponseWriter, r *http.Request, appointmentID int) {
	currentUser, err := server.getCurrentUser(r)
	if err != nil {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	filters := map[string]interface{}{
		"id": appointmentID,
	}

	appointments, err := server.AppointmentService.GetAllAppointments(currentUser, 1, 0, filters, "")
	if err != nil || len(appointments) == 0 {
		writeJSONError(w, "Appointment not found", http.StatusNotFound)
		return
	}

	appointment := appointments[0]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(appointment)
}

func (server *Server) updateAppointment(w http.ResponseWriter, r *http.Request, appointmentID int) {
	currentUser, err := server.getCurrentUser(r)
    if err != nil {
        writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

	var updatedAppointment models.Appointment
	if err := json.NewDecoder(r.Body).Decode(&updatedAppointment); err != nil {
		writeJSONError(w, "Invalid input", http.StatusBadRequest)
		return
	}

	if err := updatedAppointment.Validate(); err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	updatedAppointment.ID = appointmentID
	
	if err := server.AppointmentService.UpdateAppointment(currentUser, updatedAppointment); err != nil {
		writeJSONError(w, err.Error(), http.StatusForbidden)
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
		writeJSONError(w, "Invalid appointment ID", http.StatusBadRequest)
		return
	}

	var statusUpdate struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&statusUpdate); err != nil {
		writeJSONError(w, "Invalid input", http.StatusBadRequest)
		return
	}

	if err := server.AppointmentService.UpdateAppointmentStatus(appointmentID, statusUpdate.Status); err != nil {
		http.Error(w, "Failed to update appointment status", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (server *Server) deleteAppointment(w http.ResponseWriter, r *http.Request, appointmentID int) {
	currentUser, err := server.getCurrentUser(r)
    if err != nil {
        writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

	if err := server.AppointmentService.DeleteAppointment(currentUser, appointmentID); err != nil {
		writeJSONError(w, err.Error(), http.StatusForbidden)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}