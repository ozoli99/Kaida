package service

import "github.com/ozoli99/Kaida/models"

type UserService interface {
	RegisterUser(username, email, password, role string) (*models.User, error)
	AuthenticateUser(email, password string) (*models.User, error)
}