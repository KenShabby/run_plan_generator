package main

import (
	"net/http"

	"github.com/KenShabby/run_plan_generator/internal/db"
	"github.com/KenShabby/run_plan_generator/internal/templates/pages"
	"github.com/go-chi/chi/v5"
)

func (app *application) registerMiscRoutes(r chi.Router) {
	r.Get("/health", app.handleHealth)
	r.Get("/", app.handleHome)
}
func (app *application) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (app *application) handleHome(w http.ResponseWriter, r *http.Request) {
	user, loggedIn := userFromContext(r.Context())
	if !loggedIn {
		pages.Index(app.username(r)).Render(r.Context(), w)
		return
	}
	// Fetch HR profile and zones
	var hrProfile *db.UserHrProfile
	var hrZones []db.HrZone
	var hrHistory []db.UserHrHistory

	profile, err := app.queries.GetHRProfileByUser(r.Context(), user.ID)
	if err == nil {
		hrProfile = &profile
		hrZones, _ = app.queries.GetHRZonesByUser(r.Context(), user.ID)
		hrHistory, _ = app.queries.GetHRHistoryByUser(r.Context(), user.ID)
	}

	// Fetch next race
	var nextRace *db.GetNextRaceRow
	race, err := app.queries.GetNextRace(r.Context(), user.ID)
	if err == nil {
		nextRace = &race
	}

	// Fetch upcoming runs this week
	upcomingRuns, _ := app.queries.GetUpcomingRunsThisWeek(r.Context(), user.ID)

	if r.Header.Get("HX-Request") == "true" {
		pages.DashboardContent(hrProfile, hrZones, hrHistory, nextRace, upcomingRuns, app.username(r)).Render(r.Context(), w)
	} else {
		pages.Dashboard(hrProfile, hrZones, hrHistory, nextRace, upcomingRuns, app.username(r)).Render(r.Context(), w)
	}
}
