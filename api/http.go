package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ozoli99/Kaida/models"
	"github.com/ozoli99/Kaida/service"
)

type Server struct {
	Service *service.AppointmentService
}

func (s *Server) StartServer(port string) error {
	http.HandleFunc("/appointments", s.handleAppointments)
	http.HandleFunc("/appointments/", s.handleAppointmentByID)
	return http.ListenAndServe(":"+port, nil)
}

func (s *Server) handleAppointments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
		case http.MethodGet:
			s.getAllAppointments(w, r)
		case http.MethodPost:
			s.createAppointment(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAppointmentByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/appointments/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
		case http.MethodGet:
			s.getAppointmentByID(w, id)
		case http.MethodPut:
			s.updateAppointment(w, r, id)
		case http.MethodDelete:
			s.deleteAppointment(w, id)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getAllAppointments(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit <= 0 {
		limit = 10
	}
	offset, _ := strconv.Atoi(query.Get("offset"))
	if offset < 0 {
		offset = 0
	}

	appointments, err := s.Service.GetAll(limit, offset)
	if err != nil {
		http.Error(w, "Failed to fetch appointments", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(appointments)
}

func (s *Server) createAppointment(w http.ResponseWriter, r *http.Request) {
	var a models.Appointment
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	id, err := s.Service.Create(a)
	if err != nil {
		http.Error(w, "Failed to create appointment", http.StatusInternalServerError)
		return
	}
	a.ID = id
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(a)
}

func (s *Server) getAppointmentByID(w http.ResponseWriter, id int) {
	appointments, err := s.Service.GetAll(1, id-1)
	if err != nil || len(appointments) == 0 {
		http.Error(w, "Appointment not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(appointments[0])
}

func (s *Server) updateAppointment(w http.ResponseWriter, r *http.Request, id int) {
	var a models.Appointment
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	a.ID = id
	if err := s.Service.Update(a); err != nil {
		http.Error(w, "Failed to update appointment", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a)
}

func (s *Server) deleteAppointment(w http.ResponseWriter, id int) {
	if err := s.Service.Delete(id); err != nil {
		http.Error(w, "Failed to delete appointment", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}