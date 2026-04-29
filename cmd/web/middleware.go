package main

import (
	"net/http"
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
