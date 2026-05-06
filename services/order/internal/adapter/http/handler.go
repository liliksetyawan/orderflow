// Package http is the driving adapter for the REST API. It turns HTTP
// requests into use case calls and serializes results back. No business
// logic lives here — only translation.
package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/lilik-setyawan/orderflow/services/order/internal/app/usecase"
	"github.com/lilik-setyawan/orderflow/services/order/internal/domain"
)

type OrderHandler struct {
	create *usecase.CreateOrder
	get    *usecase.GetOrder
	log    zerolog.Logger
}

func NewOrderHandler(create *usecase.CreateOrder, get *usecase.GetOrder, log zerolog.Logger) *OrderHandler {
	return &OrderHandler{
		create: create,
		get:    get,
		log:    log.With().Str("component", "http-handler").Logger(),
	}
}

type itemDTO struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
	Price    int64  `json:"price"`
}

type createReq struct {
	CustomerID string    `json:"customer_id"`
	Items      []itemDTO `json:"items"`
}

type orderResp struct {
	ID         string    `json:"id"`
	CustomerID string    `json:"customer_id"`
	Total      int64     `json:"total"`
	Status     string    `json:"status"`
	Items      []itemDTO `json:"items"`
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}

	items := make([]domain.Item, len(req.Items))
	for i, it := range req.Items {
		items[i] = domain.Item{SKU: it.SKU, Quantity: it.Quantity, Price: it.Price}
	}

	order, err := h.create.Execute(r.Context(), usecase.CreateOrderInput{
		CustomerID: req.CustomerID,
		Items:      items,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrEmptyItems),
			errors.Is(err, domain.ErrInvalidItem),
			errors.Is(err, domain.ErrInvalidCustomer):
			writeErr(w, http.StatusBadRequest, err.Error())
		default:
			h.log.Error().Err(err).Msg("create order")
			writeErr(w, http.StatusInternalServerError, "internal error")
		}
		return
	}
	writeJSON(w, http.StatusCreated, toResp(order))
}

func (h *OrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	order, err := h.get.Execute(r.Context(), id)
	if errors.Is(err, domain.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		h.log.Error().Err(err).Msg("get order")
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, toResp(order))
}

func toResp(o *domain.Order) orderResp {
	items := make([]itemDTO, len(o.Items))
	for i, it := range o.Items {
		items[i] = itemDTO{SKU: it.SKU, Quantity: it.Quantity, Price: it.Price}
	}
	return orderResp{
		ID:         o.ID,
		CustomerID: o.CustomerID,
		Total:      o.Total,
		Status:     string(o.Status),
		Items:      items,
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
