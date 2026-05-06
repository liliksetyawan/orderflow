package domain

import "errors"

var (
	ErrInvalidInput      = errors.New("payment: invalid input")
	ErrInvalidAmount     = errors.New("payment: amount must be positive")
	ErrInvalidTransition = errors.New("payment: invalid status transition")
	ErrNotFound          = errors.New("payment: not found")
	ErrConcurrentUpdate  = errors.New("payment: concurrent update")
	ErrAlreadyExists     = errors.New("payment: already exists for order")
)
