package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/ozoli99/Kaida/models"
	"github.com/ozoli99/Kaida/service"
)

type Server struct {
	Service *service.AppointmentService
	WebSocket *WebSocketServer
	Middleware []func(http.Handler) http.Handler
	Authenticator func(r *http.Request) bool
}

func (s *Server) AddMiddleware(mw func(http.Handler) http.Handler) {
	s.Middleware = append(s.Middleware, mw)
}

func (s *Server) applyMiddleware(h http.Handler) http.Handler {
	for _, mw := range s.Middleware {
		h = mw(h)
	}
	return h
}

func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.Authenticator != nil && !s.Authenticator(r) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) StartServer(port string) error {
	http.Handle("/appointments", s.applyMiddleware(s.AuthMiddleware(http.HandlerFunc(s.handleAppointments))))
	http.Handle("/appointments/", s.applyMiddleware(s.AuthMiddleware(http.HandlerFunc(s.handleAppointmentByID))))
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

	filters := map[string]interface{}{}
	if customerName := query.Get("customer_name"); customerName != "" {
		filters["customer_name"] = customerName
	}
	if start := query.Get("start"); start != "" {
		filters["start"] = start
	}
	if end := query.Get("end"); end != "" {
		filters["end"] = end
	}

	sort := query.Get("sort")

	appointments, err := s.Service.GetAll(limit, offset, filters, sort)
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
	if err := a.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id, err := s.Service.Create(a)
	if err != nil {
		http.Error(w, "Failed to create appointment", http.StatusInternalServerError)
		return
	}
	a.ID = id
	if s.WebSocket != nil {
		message, _ := json.Marshal(a)
		s.WebSocket.Broadcast(message)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(a)
}

func (s *Server) getAppointmentByID(w http.ResponseWriter, id int) {
	filters := map[string]interface{}{
		"id": id,
	}
	appointments, err := s.Service.GetAll(1, 0, filters, "")
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
	if err := a.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.ID = id
	if err := s.Service.Update(a); err != nil {
		http.Error(w, "Failed to update appointment", http.StatusInternalServerError)
		return
	}
	if s.WebSocket != nil {
		message, _ := json.Marshal(a)
		s.WebSocket.Broadcast(message)
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

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}