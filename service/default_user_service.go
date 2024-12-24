package service

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/ozoli99/Kaida/db"
	"github.com/ozoli99/Kaida/models"
)

type DefaultUserService struct {
	Database db.Database
}

var _ UserService = (*DefaultUserService)(nil)

func (userService *DefaultUserService) RegisterUser(username, email, password, role string) (*models.User, error) {
	if strings.TrimSpace(username) == "" || strings.TrimSpace(email) == "" || strings.TrimSpace(password) == "" {
		return nil, errors.New("username, email, and password are required")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Username: username,
		Email: email,
		Password: string(hashed),
		Role: role,
	}

	err = userService.Database.CreateUser(user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (userService *DefaultUserService) AuthenticateUser(email, password string) (*models.User, error) {
	user, err := userService.Database.GetUserByEmail(email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	return user, nil
}