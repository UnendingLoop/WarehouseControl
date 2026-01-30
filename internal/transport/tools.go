package transport

import (
	"errors"

	"github.com/UnendingLoop/EventBooker/internal/model"
)

func errorCodeDefiner(err error) int {
	switch {
	case errors.Is(err, model.ErrInvalidToken),
		errors.Is(err, model.ErrInvalidCredentials),
		errors.Is(err, model.ErrIncorrectEmail),
		errors.Is(err, model.ErrIncorrectPhone),
		errors.Is(err, model.ErrIncorrectEventID),
		errors.Is(err, model.ErrIncorrectBookID),
		errors.Is(err, model.ErrIncorrectUserID),
		errors.Is(err, model.ErrIncorrectUserRole),
		errors.Is(err, model.ErrIncorrectEventTime),
		errors.Is(err, model.ErrEmptyEventInfo),
		errors.Is(err, model.ErrEmptyBookInfo),
		errors.Is(err, model.ErrEmptyEmail):
		return 400
	case errors.Is(err, model.ErrAccessDenied):
		return 403
	case errors.Is(err, model.ErrUserNotFound),
		errors.Is(err, model.ErrBookNotFound),
		errors.Is(err, model.ErrEventNotFound):
		return 404
	case errors.Is(err, model.ErrBookIsConfirmed),
		errors.Is(err, model.ErrNoSeatsAvailable),
		errors.Is(err, model.ErrExpiredEvent),
		errors.Is(err, model.ErrExpiredBook),
		errors.Is(err, model.ErrBookIsCancelled),
		errors.Is(err, model.ErrEventBusy),
		errors.Is(err, model.ErrUserAlreadyExists):
		return 409
	default:
		return 500
	}
}
