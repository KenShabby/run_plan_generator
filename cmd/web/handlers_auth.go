package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/KenShabby/run_plan_generator/internal/templates/pages"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

func (app *application) registerAuthRoutes(r chi.Router) {
	r.Get("/login", app.handleLoginForm)
	r.Post("/login", app.handleLogin)
}

func (app *application) handleLoginForm(w http.ResponseWriter, r *http.Request) {
	username := app.username(r)
	pages.Login("", username).Render(r.Context(), w)
}

func (app *application) handleLogin(w http.ResponseWriter, r *http.Request) {
	username := app.username(r)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")

	user, err := app.queries.GetUserByEmail(r.Context(), email)
	if err != nil {
		pages.Login("Invalid email or password.", username).Render(r.Context(), w)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		pages.Login("Invalid email or password.", username).Render(r.Context(), w)
		return
	}

	if err := app.setSessionUserID(w, r, user.ID); err != nil {
		log.Printf("session error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/plans", http.StatusSeeOther)
}
