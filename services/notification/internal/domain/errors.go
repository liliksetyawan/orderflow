package domain

import "errors"

var (
	ErrInvalidInput  = errors.New("notification: invalid input")
	ErrNotFound      = errors.New("notification: not found")
	ErrAlreadyExists = errors.New("notification: already sent for this order/type/channel")
)
