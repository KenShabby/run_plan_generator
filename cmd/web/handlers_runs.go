package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/KenShabby/run_plan_generator/internal/db"
	"github.com/KenShabby/run_plan_generator/internal/templates/pages"
	"github.com/go-chi/chi/v5"
)

func (app *application) registerRunRoutes(r chi.Router) {
	r.Delete("/runs/{id}", app.handleDeleteRun)
	r.Get("/runs/{id}", app.handleGetRun)
}

func (app *application) handleDeleteRun(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	userID, ok := app.getSessionUserID(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := app.queries.DeleteRunDayIfOwner(r.Context(), db.DeleteRunDayIfOwnerParams{
		ID:     int32(id),
		UserID: userID,
	}); err != nil {
		log.Printf("error deleting run: %v", err)
		http.Error(w, "failed to delete run", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (app *application) handleGetRun(w http.ResponseWriter, r *http.Request) {
	username := app.username(r)
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	userID, ok := app.getSessionUserID(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	run, err := app.queries.GetRunDayWithPlanOwner(r.Context(), int32(id))
	if err != nil {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}
	if run.UserID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	segments, err := app.queries.ListSegmentsByRun(r.Context(), int32(id))
	if err != nil {
		log.Printf("error fetching segments: %v", err)
		http.Error(w, "failed to load segments", http.StatusInternalServerError)
		return
	}

	// fetch user's HR zones and build a lookup map
	hrrZones, err := app.queries.GetHRZonesByUser(r.Context(), userID)
	if err != nil {
		log.Printf("error fetching hr zones: %v", err)
	}
	zoneMap := make(map[int32]db.HrZone)
	for _, z := range hrrZones {
		zoneMap[z.ZoneNumber] = z
	}

	// convert to db.RunDay for the template
	runDay := db.RunDay{
		ID:            run.ID,
		PlanID:        run.PlanID,
		Date:          run.Date,
		RunType:       run.RunType,
		TotalDistance: run.TotalDistance,
		TotalDuration: run.TotalDuration,
		Completed:     run.Completed,
		Notes:         run.Notes,
		CreatedAt:     run.CreatedAt,
	}

	if r.Header.Get("HX-Request") == "true" {
		pages.RunDetailContent(runDay, segments, zoneMap).Render(r.Context(), w)
	} else {
		pages.RunDetail(runDay, segments, username, zoneMap).Render(r.Context(), w)
	}
}
