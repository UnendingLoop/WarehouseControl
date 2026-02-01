package model

import "errors"

var (
	// 404
	ErrUserNotFound = errors.New("requested user id not found")
	ErrItemNotFound = errors.New("requested item id not found")

	// 400
	ErrInvalidToken       = errors.New("invalid auth-token provided")
	ErrInvalidCredentials = errors.New("email or password is incorrect")

	ErrInvalidOrderBy      = errors.New("invalid ordering parameter specified")
	ErrInvalidAscDesc      = errors.New("invalid ASC/DESC provided")
	ErrInvalidStartEndTime = errors.New("invalid start/end time provided: start cannot be later than end")
	ErrInvalidPage         = errors.New("invalid page value provided: value must be > 0")
	ErrInvalidLimit        = errors.New("invalid limit value provided: value must be > 0 and < 1000")

	ErrIncorrectItemID   = errors.New("incorrect item id provided")
	ErrIncorrectUserName = errors.New("incorrect username provided")
	ErrIncorrectUserRole = errors.New("incorrect user role is provided")
	ErrEmptyItemInfo     = errors.New("incomplete data provided to create item")
	ErrEmptyTitle        = errors.New("invalid item title provided")
	ErrInvalidPrice      = errors.New("invalid item price provided")
	ErrInvalidAvail      = errors.New("invalid item available amount provided")
	ErrNoFieldsToUpdate  = errors.New("nothing to update in item")

	// 403
	ErrAccessDenied = errors.New("lack permissions to complete operation")

	// 500
	ErrCommon500 = errors.New("something went wrong. Try again later")

	// 409
	ErrUserAlreadyExists = errors.New("user with such username already exists")
)
