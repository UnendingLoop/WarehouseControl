package model

import "errors"

var (
	// 404
	ErrUserNotFound  = errors.New("requested user id not found")
	ErrBookNotFound  = errors.New("requested booking id not found")
	ErrEventNotFound = errors.New("requested event id not found")

	// 400
	ErrInvalidToken       = errors.New("invalid auth-token provided")
	ErrInvalidCredentials = errors.New("email or password is incorrect")

	ErrIncorrectEmail     = errors.New("incorrect email provided")
	ErrIncorrectPhone     = errors.New("incorrect telephone number provided")
	ErrIncorrectEventID   = errors.New("incorrect event id provided")
	ErrIncorrectBookID    = errors.New("incorrect booking id provided")
	ErrIncorrectUserID    = errors.New("incorrect user id provided")
	ErrIncorrectUserRole  = errors.New("incorrect user role is provided")
	ErrIncorrectEventTime = errors.New("event date cannot be in the past")
	ErrEmptyEventInfo     = errors.New("incomplete data provided to create event")
	ErrEmptyBookInfo      = errors.New("incomplete data provided to book event")
	ErrEmptyEmail         = errors.New("empty email provided")

	// 403
	ErrAccessDenied = errors.New("you don't have enough permissions to complete this operation")

	// 500
	ErrCommon500 = errors.New("something went wrong. Try again later")

	// 409
	ErrBookIsConfirmed   = errors.New("requested booking is already confirmed")
	ErrNoSeatsAvailable  = errors.New("no more seats to book for this event")
	ErrExpiredEvent      = errors.New("the event you are trying to book has expired")
	ErrExpiredBook       = errors.New("requested booking confirmation deadline has expired")
	ErrBookIsCancelled   = errors.New("requested booking is already cancelled")
	ErrEventBusy         = errors.New("requested event not available for deletion. Remove confirmed bookings first")
	ErrUserAlreadyExists = errors.New("user with such email already exists")
)
