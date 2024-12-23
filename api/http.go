package api

import (
	"log"
	"net/http"

	"github.com/ozoli99/Kaida/service"
)

type Server struct {
	AppointmentService *service.AppointmentService
	UserService        *service.UserService
	WebSocketServer    *WebSocketServer
	MiddlewareChain    []func(http.Handler) http.Handler
}

func (server *Server) AddMiddleware(middleware func(http.Handler) http.Handler) {
	server.MiddlewareChain = append(server.MiddlewareChain, middleware)
}

func (server *Server) applyMiddleware(handler http.Handler) http.Handler {
	for _, middleware := range server.MiddlewareChain {
		handler = middleware(handler)
	}
	return handler
}

func (server *Server) StartServer(port string) error {
	http.Handle("/appointments", server.applyMiddleware(http.HandlerFunc(server.handleAppointments)))
	http.Handle("/appointments/", server.applyMiddleware(http.HandlerFunc(server.handleAppointmentByID)))
	http.Handle("/appointments/status/", server.applyMiddleware(http.HandlerFunc(server.updateAppointmentStatus)))
	http.Handle("/recurring", server.applyMiddleware(http.HandlerFunc(server.handleRecurringAppointments)))
	
	http.HandleFunc("/users/register", server.handleUserRegister)
    http.HandleFunc("/users/login", server.handleUserLogin)
	return http.ListenAndServe(":"+port, nil)
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