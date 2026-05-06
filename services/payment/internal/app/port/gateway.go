package port

import (
	"context"
	"errors"
)

// ErrDeclined indicates the gateway rejected the charge for a business
// reason (insufficient funds, fraud rule, etc). Adapters wrap this when
// the result is a hard "no" from the provider. Errors *not* wrapping
// ErrDeclined are treated as transient by the use case.
var ErrDeclined = errors.New("payment gateway: declined")

// PaymentGateway is the outbound port for the payment provider integration.
type PaymentGateway interface {
	// Charge attempts to capture amount for the given order. Returns the
	// provider-side charge id on success, ErrDeclined-wrapped error on
	// hard failure, or any other error for transient issues.
	Charge(ctx context.Context, orderID, customerID string, amount int64) (providerRef string, err error)

	// Refund reverses a previously authorized charge. Idempotent at the
	// gateway level (real gateways like Stripe key by providerRef).
	Refund(ctx context.Context, providerRef string, amount int64) error
}
