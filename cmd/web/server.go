package main

import (
	"net/http"

	"github.com/KenShabby/run_plan_generator/internal/templates/pages"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func newServer() http.Handler {
	r := chi.NewRouter()

	// Basic middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// root
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		pages.Index().Render(r.Context(), w)
	})

	r.Get("/plans", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("HX-Request") == "true" {
			// HTMX request - return just the inner content
			pages.PlansContent(nil).Render(r.Context(), w)
		} else {
			// Full page load - return the whole layout
			pages.Plans(nil).Render(r.Context(), w)
		}
	})

	return r

}
