package domain

import "errors"

var (
	ErrInvalidInput      = errors.New("inventory: invalid input")
	ErrEmptyItems        = errors.New("inventory: items required")
	ErrInvalidItem       = errors.New("inventory: invalid item")
	ErrInvalidTransition = errors.New("inventory: invalid status transition")
	ErrNotFound          = errors.New("inventory: not found")
	ErrConcurrentUpdate  = errors.New("inventory: concurrent update")
	ErrAlreadyExists     = errors.New("inventory: reservation already exists for order")
	ErrInsufficientStock = errors.New("inventory: insufficient stock")
)
