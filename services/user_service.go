package services

import "github.com/AbaraEmmanuel/jaromind-backend/models"

type UserService interface {
	Register(student models.User) error
	Login(email, password string) (string, error)
}
