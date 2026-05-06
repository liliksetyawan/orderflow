// Package idgen is the driven adapter for port.IDGenerator using UUID v7
// (time-ordered, friendly to btree indexes).
package idgen

import (
	"github.com/google/uuid"

	"github.com/lilik-setyawan/orderflow/services/payment/internal/app/port"
)

type UUIDv7 struct{}

var _ port.IDGenerator = (*UUIDv7)(nil)

func (UUIDv7) New() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}
