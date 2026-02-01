package transport

import (
	"errors"

	"github.com/UnendingLoop/WarehouseControl/internal/model"
)

func errorCodeDefiner(err error) int {
	switch {
	case errors.Is(err, model.ErrInvalidToken),
		errors.Is(err, model.ErrInvalidCredentials),
		errors.Is(err, model.ErrInvalidOrderBy),
		errors.Is(err, model.ErrInvalidAscDesc),
		errors.Is(err, model.ErrInvalidStartEndTime),
		errors.Is(err, model.ErrInvalidPage),
		errors.Is(err, model.ErrInvalidLimit),
		errors.Is(err, model.ErrIncorrectItemID),
		errors.Is(err, model.ErrIncorrectUserName),
		errors.Is(err, model.ErrIncorrectUserRole),
		errors.Is(err, model.ErrEmptyItemInfo),
		errors.Is(err, model.ErrEmptyTitle),
		errors.Is(err, model.ErrInvalidPrice),
		errors.Is(err, model.ErrInvalidAvail),
		errors.Is(err, model.ErrNoFieldsToUpdate):
		return 400
	case errors.Is(err, model.ErrAccessDenied):
		return 403
	case errors.Is(err, model.ErrUserNotFound),
		errors.Is(err, model.ErrItemNotFound):
		return 404
	case errors.Is(err, model.ErrUserAlreadyExists):
		return 409
	default:
		return 500
	}
}
