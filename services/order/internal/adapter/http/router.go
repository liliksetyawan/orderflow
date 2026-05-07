package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func NewRouter(orders *OrderHandler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	// CORS: allow the local Vite dev server and any localhost variant a
	// developer might reach for. Production should narrow this list.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:4173", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Idempotency-Key"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", health)
	r.Get("/ready", health)

	r.Route("/v1/orders", func(r chi.Router) {
		r.Post("/", orders.Create)
		r.Get("/", orders.List)
		r.Get("/{id}", orders.Get)
	})

	return r
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
