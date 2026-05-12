package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/KenShabby/run_plan_generator/internal/db"
	"github.com/KenShabby/run_plan_generator/internal/templates/pages"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

func (app *application) registerAuthRoutes(r chi.Router) {
	r.Get("/login", app.handleLoginForm)
	r.Post("/login", app.handleLogin)
	r.Get("/register", app.handleGetRegister)
	r.Post("/register", app.handlePostRegister)
	r.Post("/logout", app.handleLogout)
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

func (app *application) handleGetRegister(w http.ResponseWriter, r *http.Request) {
	username := app.username(r)
	pages.Register("", username).Render(r.Context(), w)
}

// POST /register — handle submission
func (app *application) handlePostRegister(w http.ResponseWriter, r *http.Request) {
	navUsername := app.username(r)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	newUsername := strings.TrimSpace(r.FormValue("username"))
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	confirm := r.FormValue("confirm_password")

	if newUsername == "" || email == "" || password == "" {
		pages.Register("All fields are required.", navUsername).Render(r.Context(), w)
		return
	}
	if password != confirm {
		pages.Register("Passwords do not match.", navUsername).Render(r.Context(), w)
		return
	}
	if len(password) < 8 {
		pages.Register("Password must be at least 8 characters.", navUsername).Render(r.Context(), w)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("bcrypt error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	user, err := app.queries.CreateUser(r.Context(), db.CreateUserParams{
		Username:     newUsername,
		Email:        email,
		PasswordHash: string(hash),
	})
	if err != nil {
		if strings.Contains(err.Error(), "23505") {
			pages.Register("That username or email is already taken.", navUsername).Render(r.Context(), w)
			return
		}
		log.Printf("CreateUser error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := app.setSessionUserID(w, r, user.ID); err != nil {
		log.Printf("session error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/plans", http.StatusSeeOther)
}

func (app *application) handleLogout(w http.ResponseWriter, r *http.Request) {
	app.clearSession(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
