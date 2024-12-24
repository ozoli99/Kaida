package api

import (
	"encoding/json"
	"net/http"

	"github.com/ozoli99/Kaida/models"
)

func (server *Server) handleUserRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Email string `json:"email"`
		Password string `json:"password"`
		Role string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := server.UserService.RegisterUser(
		req.Username, 
		req.Email, 
		req.Password, 
		req.Role,
	)
    if err != nil {
        writeJSONError(w, err.Error(), http.StatusBadRequest)
        return
    }

    user.Password = ""
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}

func (server *Server) handleUserLogin(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req struct {
        Email    string `json:"email"`
        Password string `json:"password"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeJSONError(w, err.Error(), http.StatusBadRequest)
        return
    }

    user, err := server.UserService.AuthenticateUser(req.Email, req.Password)
    if err != nil {
        writeJSONError(w, "Invalid credentials", http.StatusUnauthorized)
        return
    }

    user.Password = ""
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}

func (server *Server) getCurrentUser(r *http.Request) (*models.User, error) {
    user := &models.User{
        ID:    2,
        Role:  "customer",
        Email: "test@example.com",
    }
    return user, nil
}

func writeJSONError(w http.ResponseWriter, message string, status int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]string{"error": message})
}