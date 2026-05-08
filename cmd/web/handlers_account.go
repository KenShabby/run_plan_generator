package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/KenShabby/run_plan_generator/internal/db"
	"github.com/KenShabby/run_plan_generator/internal/templates/pages"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (app *application) registerAccountRoutes(r chi.Router) {
	r.Get("/account", app.handleGetAccount)
	r.Get("/account/hr", app.handleGetAccountHr)
	r.Post("/account/hr", app.handlePostAccountHr)
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
