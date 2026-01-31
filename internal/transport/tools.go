package transport

import (
	"errors"

	"github.com/UnendingLoop/WarehouseControl/internal/model"
)

func errorCodeDefiner(err error) int {
	switch {
	case errors.Is(err, model.ErrInvalidToken),
		errors.Is(err, model.ErrInvalidCredentials),
		errors.Is(err, model.ErrIncorrectUserRole):
		return 400
	case errors.Is(err, model.ErrAccessDenied):
		return 403
	case errors.Is(err, model.ErrUserNotFound),
		errors.Is(err, model.ErrItemNotFound):
		return 404
	case errors.Is(err, model.ErrEventBusy),
		errors.Is(err, model.ErrUserAlreadyExists):
		return 409
	default:
		return 500
	}
}
