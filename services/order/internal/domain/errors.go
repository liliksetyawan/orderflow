package domain

import "errors"

var (
	ErrEmptyItems        = errors.New("order: items required")
	ErrInvalidItem       = errors.New("order: invalid item")
	ErrInvalidCustomer   = errors.New("order: customer_id required")
	ErrInvalidTransition = errors.New("order: invalid status transition")
	ErrNotFound          = errors.New("order: not found")
	ErrConcurrentUpdate  = errors.New("order: concurrent update")
)
