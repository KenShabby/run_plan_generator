package main

import (
	"net/http"
	"os"

	"github.com/gorilla/csrf"
)

func (app *application) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := app.getSessionUserID(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (app *application) loadUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := app.getSessionUserID(r)
		if ok {
			user, err := app.queries.GetUserByID(r.Context(), id)
			if err == nil {
				r = r.WithContext(withUser(r.Context(), user))
			}
		}
		next.ServeHTTP(w, r)
	})
}

func plaintextHTTPInDev(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("ENV") != "production" {
			r = csrf.PlaintextHTTPRequest(r)
		}
		next.ServeHTTP(w, r)
	})
}
