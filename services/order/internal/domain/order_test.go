package domain_test

import (
	"errors"
	"testing"

	"github.com/lilik-setyawan/orderflow/services/order/internal/domain"
)

func items() []domain.Item {
	return []domain.Item{{SKU: "A", Quantity: 2, Price: 1000}}
}

func TestNewOrder_HappyPath(t *testing.T) {
	o, err := domain.NewOrder("id1", "c1", items())
	if err != nil {
		t.Fatal(err)
	}
	if o.Total != 2000 {
		t.Errorf("total = %d, want 2000", o.Total)
	}
	if o.Status != domain.StatusPending {
		t.Errorf("status = %s, want PENDING", o.Status)
	}
	if o.Version != 1 {
		t.Errorf("version = %d, want 1", o.Version)
	}
}

func TestNewOrder_RejectsEmptyItems(t *testing.T) {
	_, err := domain.NewOrder("id1", "c1", nil)
	if !errors.Is(err, domain.ErrEmptyItems) {
		t.Errorf("err = %v, want ErrEmptyItems", err)
	}
}

func TestNewOrder_RejectsBadItem(t *testing.T) {
	cases := map[string][]domain.Item{
		"empty sku":     {{SKU: "", Quantity: 1, Price: 100}},
		"zero qty":      {{SKU: "A", Quantity: 0, Price: 100}},
		"negative qty":  {{SKU: "A", Quantity: -1, Price: 100}},
		"negative price": {{SKU: "A", Quantity: 1, Price: -1}},
	}
	for name, its := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := domain.NewOrder("id", "c1", its)
			if !errors.Is(err, domain.ErrInvalidItem) {
				t.Errorf("err = %v, want ErrInvalidItem", err)
			}
		})
	}
}

func TestNewOrder_RejectsEmptyCustomer(t *testing.T) {
	_, err := domain.NewOrder("id", "", items())
	if !errors.Is(err, domain.ErrInvalidCustomer) {
		t.Errorf("err = %v, want ErrInvalidCustomer", err)
	}
}

func TestStateMachine_HappyPath(t *testing.T) {
	o, _ := domain.NewOrder("id1", "c1", items())
	if err := o.MarkAuthorized(); err != nil {
		t.Fatal(err)
	}
	if o.Status != domain.StatusAuthorized {
		t.Errorf("status = %s, want AUTHORIZED", o.Status)
	}
	if err := o.MarkConfirmed(); err != nil {
		t.Fatal(err)
	}
	if o.Status != domain.StatusConfirmed {
		t.Errorf("status = %s, want CONFIRMED", o.Status)
	}
}

func TestStateMachine_RejectsSkippedTransitions(t *testing.T) {
	o, _ := domain.NewOrder("id1", "c1", items())
	if err := o.MarkConfirmed(); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Errorf("PENDING→CONFIRMED: err = %v, want ErrInvalidTransition", err)
	}
}

func TestStateMachine_CancelFromConfirmedRejected(t *testing.T) {
	o, _ := domain.NewOrder("id1", "c1", items())
	_ = o.MarkAuthorized()
	_ = o.MarkConfirmed()
	if err := o.Cancel(); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Errorf("CONFIRMED→CANCELED: err = %v, want ErrInvalidTransition", err)
	}
}

func TestStateMachine_CancelFromAuthorized(t *testing.T) {
	o, _ := domain.NewOrder("id1", "c1", items())
	_ = o.MarkAuthorized()
	if err := o.Cancel(); err != nil {
		t.Fatal(err)
	}
	if o.Status != domain.StatusCanceled {
		t.Errorf("status = %s, want CANCELED", o.Status)
	}
}
