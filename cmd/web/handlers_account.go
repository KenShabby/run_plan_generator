package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/KenShabby/run_plan_generator/internal/db"
	"github.com/KenShabby/run_plan_generator/internal/templates/pages"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (app *application) registerAccountRoutes(r chi.Router) {
	r.Get("/account", app.handleGetAccount)
	r.Get("/account/hr", app.handleGetAccountHr)
	r.Post("/account/hr", app.handlePostAccountHr)
	r.Post("/account/username", app.handlePostAccountUsername)
	r.Post("/account/email", app.handlePostAccountEmail)
	r.Post("/account/password", app.handlePostAccountPassword)
	r.Post("/account/delete", app.handlePostAccountDelete)
}

func (app *application) handleGetAccount(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	pages.Account(user, "", "", "", app.username(r)).Render(r.Context(), w)
}

func (app *application) handleGetAccountHr(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	profile, err := app.queries.GetHRProfileByUser(r.Context(), user.ID)
	if err != nil {
		// no profile yet, render empty form
		pages.HRProfile(nil, nil, "", app.username(r)).Render(r.Context(), w)
		return
	}

	method := bestMethod(
		int(profile.MaxHr.Int32),
		int(profile.RestingHr.Int32),
		int(profile.LactateThresholdHr.Int32),
	)
	zones := calculateZones(
		int(profile.MaxHr.Int32),
		int(profile.RestingHr.Int32),
		int(profile.LactateThresholdHr.Int32),
		method,
	)

	pages.HRProfile(&profile, zones, "", app.username(r)).Render(r.Context(), w)

}

func (app *application) handlePostAccountHr(w http.ResponseWriter, r *http.Request) {

	user, ok := userFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	maxHR, err := strconv.Atoi(r.FormValue("max_hr"))
	if err != nil || maxHR < 100 || maxHR > 220 {
		pages.HRProfile(nil, nil, "Max heart rate must be between 100 and 220.", app.username(r)).Render(r.Context(), w)
		return
	}

	var restingHR, lthr int
	if v := r.FormValue("resting_hr"); v != "" {
		restingHR, _ = strconv.Atoi(v)
	}
	if v := r.FormValue("lactate_threshold_hr"); v != "" {
		lthr, _ = strconv.Atoi(v)
	}

	method := bestMethod(maxHR, restingHR, lthr)

	var restingHRVal, lthrVal pgtype.Int4
	if restingHR > 0 {
		restingHRVal = pgtype.Int4{Int32: int32(restingHR), Valid: true}
	}
	if lthr > 0 {
		lthrVal = pgtype.Int4{Int32: int32(lthr), Valid: true}
	}

	// upsert — try update first, then create
	var profile db.UserHrProfile
	existing, err := app.queries.GetHRProfileByUser(r.Context(), user.ID)
	if err != nil {
		// no profile yet, create one
		profile, err = app.queries.CreateHRProfile(r.Context(), db.CreateHRProfileParams{
			UserID:             user.ID,
			MaxHr:              pgtype.Int4{Int32: int32(maxHR), Valid: true},
			RestingHr:          restingHRVal,
			LactateThresholdHr: lthrVal,
			CalculationMethod:  method,
		})
		if err != nil {
			log.Printf("error creating hr profile: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	} else {
		profile, err = app.queries.UpdateHRProfile(r.Context(), db.UpdateHRProfileParams{
			UserID:             user.ID,
			MaxHr:              pgtype.Int4{Int32: int32(maxHR), Valid: true},
			RestingHr:          restingHRVal,
			LactateThresholdHr: lthrVal,
			CalculationMethod:  method,
		})
		if err != nil {
			log.Printf("error updating hr profile: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		// delete old zones so we can recalculate
		if err := app.queries.DeleteHRZonesByProfile(r.Context(), existing.ID); err != nil {
			log.Printf("error deleting old zones: %v", err)
		}
	}

	// Record HR history
	app.queries.InsertHRHistory(r.Context(), db.InsertHRHistoryParams{
		UserID:    user.ID,
		MaxHr:     pgtype.Int4{Int32: int32(maxHR), Valid: true},
		RestingHr: restingHRVal,
		Lthr:      lthrVal,
		Method:    method,
	})

	// calculate and save zones
	zones := calculateZones(maxHR, restingHR, lthr, method)
	for _, zone := range zones {
		_, err := app.queries.CreateHRZone(r.Context(), db.CreateHRZoneParams{
			ProfileID:   profile.ID,
			ZoneNumber:  int32(zone.Number),
			Name:        pgtype.Text{String: zone.Name, Valid: true},
			HrMin:       int32(zone.Min),
			HrMax:       int32(zone.Max),
			Description: pgtype.Text{String: zone.Description, Valid: true},
		})
		if err != nil {
			log.Printf("error saving zone: %v", err)
		}
	}

	pages.HRProfile(&profile, zones, "", app.username(r)).Render(r.Context(), w)
}

func (app *application) handlePostAccountUsername(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	newUsername := strings.TrimSpace(r.FormValue("username"))
	if newUsername == "" {
		pages.Account(user, "Username cannot be empty.", "", "", app.username(r)).Render(r.Context(), w)
		return
	}
	updated, err := app.queries.UpdateUsername(r.Context(), db.UpdateUsernameParams{
		ID:       user.ID,
		Username: newUsername,
	})
	if err != nil {
		if strings.Contains(err.Error(), "23505") {
			pages.Account(user, "That username is already taken.", "", "", app.username(r)).Render(r.Context(), w)
			return
		}
		log.Printf("error updating username: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	pages.Account(updated, "", "", "", updated.Username).Render(r.Context(), w)
}

func (app *application) handlePostAccountEmail(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	newEmail := strings.TrimSpace(r.FormValue("email"))
	if newEmail == "" {
		pages.Account(user, "", "Email cannot be empty.", "", app.username(r)).Render(r.Context(), w)
		return
	}
	updated, err := app.queries.UpdateEmail(r.Context(), db.UpdateEmailParams{
		ID:    user.ID,
		Email: newEmail,
	})
	if err != nil {
		if strings.Contains(err.Error(), "23505") {
			pages.Account(user, "", "That email is already taken.", "", app.username(r)).Render(r.Context(), w)
			return
		}
		log.Printf("error updating email: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	pages.Account(updated, "", "", "", updated.Username).Render(r.Context(), w)
}

func (app *application) handlePostAccountPassword(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	confirm := r.FormValue("confirm_password")

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		pages.Account(user, "", "", "Current password is incorrect.", app.username(r)).Render(r.Context(), w)
		return
	}
	if newPassword != confirm {
		pages.Account(user, "", "", "Passwords do not match.", app.username(r)).Render(r.Context(), w)
		return
	}
	if len(newPassword) < 8 {
		pages.Account(user, "", "", "Password must be at least 8 characters.", app.username(r)).Render(r.Context(), w)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("bcrypt error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := app.queries.UpdatePassword(r.Context(), db.UpdatePasswordParams{
		ID:           user.ID,
		PasswordHash: string(hash),
	}); err != nil {
		log.Printf("error updating password: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	pages.Account(user, "", "", "", app.username(r)).Render(r.Context(), w)
}

func (app *application) handlePostAccountDelete(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := app.queries.DeleteUser(r.Context(), user.ID); err != nil {
		log.Printf("error deleting user: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := app.clearSession(w, r); err != nil {
		log.Printf("error clearing session: %v", err)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
