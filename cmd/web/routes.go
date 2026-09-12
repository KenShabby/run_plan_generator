package main

import (
	"net/http"
	"time"

	"github.com/gorilla/csrf"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"

	"github.com/KenShabby/run_plan_generator/internal/csrfctx"
)

func withCSRFContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := csrfctx.WithToken(r.Context(), csrf.Token(r))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func clientIPKey(r *http.Request) (string, error) {
	return httprate.CanonicalizeIP(middleware.GetClientIP(r.Context())), nil
}

func newServer(app *application) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger, middleware.Recoverer, app.loadUser, withCSRFContext)
	r.Use(middleware.ClientIPFromXFF("172.16.0.0/12")) // adjust to your docker network

	// Global backstop
	r.Use(httprate.LimitBy(300, time.Minute, clientIPKey))

	app.registerMiscRoutes(r)

	// Tighter limit for auth routes
	r.Group(func(r chi.Router) {
		r.Use(httprate.LimitBy(
			10,
			time.Minute,
			clientIPKey,
		))
		app.registerAuthRoutes(r)
	})

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	r.Group(func(r chi.Router) {
		r.Use(app.requireAuth)
		app.registerAccountRoutes(r)
		app.registerPlanRoutes(r)
		app.registerRunRoutes(r)
		app.registerTemplateRoutes(r)
		app.registerActivityRoutes(r)
	})

	return r
}
