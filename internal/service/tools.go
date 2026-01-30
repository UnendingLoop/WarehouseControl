package service

import (
	"regexp"
	"strings"
	"time"

	"github.com/UnendingLoop/EventBooker/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func validateNormalizeUser(u *model.User) error {
	// Проверка роли
	if _, exists := model.RolesMap[u.Role]; !exists {
		return model.ErrIncorrectUserRole
	}
	u.UserName = strings.TrimSpace(u.UserName)
	u.UserName = strings.ToLower(u.UserName)
	matchEmail := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !matchEmail.MatchString(u.UserName) {
		return model.ErrIncorrectEmail
	}

	// Генерация хэша из пароля
	passHash, _ := bcrypt.GenerateFromPassword([]byte(u.PassHash), bcrypt.DefaultCost)
	u.PassHash = string(passHash)

	return nil
}

func validateNormalizeItem(event *model.Item) error {
	if event.Title == "" || event.TotalSeats <= 0 || event.BookWindow <= 0 {
		return model.ErrEmptyEventInfo
	}
	if event.EventDate.UTC().Before(time.Now().UTC()) {
		return model.ErrIncorrectEventTime
	}
	now := time.Now().UTC()
	event.Created = &now
	event.AvailSeats = event.TotalSeats
	event.Status = model.EventStatusActual

	return nil
}
