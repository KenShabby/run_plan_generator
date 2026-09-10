package main

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

func newServer(app *application) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger, middleware.Recoverer, app.loadUser)

	// Global rate limiter
	r.Use(httprate.LimitByIP(300, time.Minute, httprate.WithKeyFuncs(httprate.KeyByRealIP)))

	app.registerMiscRoutes(r)

	// Tighter rate limits for Auth routes
	r.Group(func(r chi.Router) {
		r.Use(httprate.LimitByIP(
			10, 
			time.Minute, 
			httprate.WithKeyFuncs(httprate.KeyByRealIP),
			httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte("Too many attempts. Try again in a minute."))
			}),
		))
	}

	app.registerAuthRoutes(r)

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Protected groups - auth required
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
