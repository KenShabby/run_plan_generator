package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func newServer(app *application) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger, middleware.Recoverer, app.loadUser)

	// Public routes, no auth required
	r.Get("/", app.handleHome)
	r.Get("/health", app.handleHealth)
	app.registerAuthRoutes(r)

	// Protected groups - auth required
	r.Group(func(r chi.Router) {
		r.Use(app.requireAuth)
		app.registerAccountRoutes(r)
		app.registerPlanRoutes(r)
		app.registerRunRoutes(r)
		app.registerTemplateRoutes(r)

		r.Post("/logout", func(w http.ResponseWriter, r *http.Request) {
			app.clearSession(w, r)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
		})
	})

	return r

}
