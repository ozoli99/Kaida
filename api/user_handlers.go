package api

import (
	"encoding/json"
	"net/http"
)

func (server *Server) handleUserRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Email string `json:"email"`
		Password string `json:"password"`
		Role string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := server.UserService.RegisterUser(req.Username, req.Email, req.Password, req.Role)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Return the created user without the password
    user.Password = ""
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}

func (server *Server) handleUserLogin(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req struct {
        Email    string `json:"email"`
        Password string `json:"password"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    user, err := server.UserService.AuthenticateUser(req.Email, req.Password)
    if err != nil {
        http.Error(w, "Invalid credentials", http.StatusUnauthorized)
        return
    }

    // This is where you'd set a session cookie or return a JWT
    // For now, just return the user minus password
    user.Password = ""
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}