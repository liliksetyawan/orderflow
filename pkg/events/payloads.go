package events

// Cross-service payload structs. Keeping them here (transport layer, not
// inside any one service) makes them the single source of truth for the
// JSON shape on the wire. Add a new field? Make sure every consumer that
// JSON-decodes the relevant message handles the absence gracefully.

type OrderLine struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
	Price    int64  `json:"price"`
}

// --- Order service emits ---

type OrderCreatedPayload struct {
	OrderID    string      `json:"order_id"`
	CustomerID string      `json:"customer_id"`
	Items      []OrderLine `json:"items"`
	Total      int64       `json:"total"`
}

type OrderConfirmedPayload struct {
	OrderID    string `json:"order_id"`
	CustomerID string `json:"customer_id"`
}

type OrderCanceledPayload struct {
	OrderID    string `json:"order_id"`
	CustomerID string `json:"customer_id"`
	Reason     string `json:"reason"`
}

// Saga commands the order service publishes.
type AuthorizePaymentPayload struct {
	OrderID    string `json:"order_id"`
	CustomerID string `json:"customer_id"`
	Amount     int64  `json:"amount"`
}

type ReserveInventoryPayload struct {
	OrderID string      `json:"order_id"`
	Items   []OrderLine `json:"items"`
}

type ReleasePaymentPayload struct {
	OrderID string `json:"order_id"`
	Amount  int64  `json:"amount"`
}

type ReleaseInventoryPayload struct {
	OrderID string `json:"order_id"`
}

// --- Replies the order saga consumes ---

type PaymentAuthorizedPayload struct {
	OrderID   string `json:"order_id"`
	PaymentID string `json:"payment_id"`
}

type PaymentFailedPayload struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

type PaymentReleasedPayload struct {
	OrderID string `json:"order_id"`
}

type InventoryReservedPayload struct {
	OrderID       string `json:"order_id"`
	ReservationID string `json:"reservation_id"`
}

type InventoryFailedPayload struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

type InventoryReleasedPayload struct {
	OrderID string `json:"order_id"`
}
