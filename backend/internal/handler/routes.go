package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func NewRouter(h *HTTPHandler, sseHub *SSEHub) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"UP"}`))
	})

	r.Route("/api", func(r chi.Router) {
		r.Route("/sync", func(r chi.Router) {
			r.Post("/start", h.StartSync)
			r.Post("/resume", h.ResumeSync)
			r.Post("/simulate-interrupt", h.SimulateInterruption)
			r.Get("/status", h.GetStatus)
			r.Get("/events", sseHub.ServeHTTP)
			r.Get("/changes", h.GetChanges)
			r.Get("/issues", h.GetIssues)
			r.Post("/issues/{id}/resolve", h.ResolveIssue)
		})

		r.Get("/products", h.ListProducts)
		r.Post("/catalog/reset", h.ResetCatalog)
	})

	return r
}
