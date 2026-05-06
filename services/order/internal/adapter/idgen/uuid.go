// Package idgen is the driven adapter for port.IDGenerator. UUID v7 gives
// us time-ordered, lexicographically-sortable ids — friendly to btree
// indexes on primary keys.
package idgen

import (
	"github.com/google/uuid"

	"github.com/liliksetyawan/orderflow/services/order/internal/app/port"
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
