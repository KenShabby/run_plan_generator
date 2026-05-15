package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/KenShabby/run_plan_generator/internal/db"
	"github.com/KenShabby/run_plan_generator/internal/models"
	"github.com/KenShabby/run_plan_generator/internal/templates/pages"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (app *application) registerActivityRoutes(r chi.Router) {
	r.Get("/activity/new", app.handleGetActivityForm)
	r.Post("/activity", app.handlePostActivity)
	r.Get("/activity/{id}", app.handleGetActivity)
	r.Get("/activity/{id}/edit", app.handleGetEditActivity)
	r.Post("/activity/{id}", app.handlePostEditActivity)
	r.Delete("/activity/{id}", app.handleDeleteActivity)
	// Log a specific planned run as complete
	r.Get("/runs/{id}/log", app.handleGetLogRun)
	r.Post("/runs/{id}/log", app.handlePostLogRun)
}

// handleGetLogRun shows the log form pre-populated from a planned run
func (app *application) handleGetLogRun(w http.ResponseWriter, r *http.Request) {
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

	// Check if already logged
	existing, err := app.queries.GetActivityLogByRunDay(r.Context(), pgtype.Int4{Int32: int32(id), Valid: true})
	if err == nil {
		// Already logged — redirect to the existing entry
		http.Redirect(w, r, fmt.Sprintf("/activity/%d", existing.ID), http.StatusSeeOther)
		return
	}

	// Pre-populate from planned run
	runDay := db.RunDay{
		ID:            run.ID,
		PlanID:        run.PlanID,
		Date:          run.Date,
		RunType:       run.RunType,
		TotalDistance: run.TotalDistance,
		TotalDuration: run.TotalDuration,
		Notes:         run.Notes,
	}

	if r.Header.Get("HX-Request") == "true" {
		pages.ActivityFormContent(runDay, models.RPEOptions, app.username(r)).Render(r.Context(), w)
	} else {
		pages.ActivityForm(runDay, models.RPEOptions, app.username(r)).Render(r.Context(), w)
	}
}

// handlePostLogRun creates an activity log entry for a planned run
func (app *application) handlePostLogRun(w http.ResponseWriter, r *http.Request) {
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

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	params, err := app.parseActivityForm(r, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	params.RunDayID = pgtype.Int4{Int32: int32(id), Valid: true}
	params.RunType = run.RunType

	_, err = app.queries.CreateActivityLog(r.Context(), params)
	if err != nil {
		app.logger.Printf("error creating activity log: %v", err)
		http.Error(w, "failed to log activity", http.StatusInternalServerError)
		return
	}

	// Mark the planned run as completed
	if err := app.queries.MarkRunDayCompleted(r.Context(), int32(id)); err != nil {
		app.logger.Printf("error marking run completed: %v", err)
	}

	http.Redirect(w, r, fmt.Sprintf("/runs/%d", id), http.StatusSeeOther)
}

// handleGetActivityForm shows a blank form for logging an unscheduled run
func (app *application) handleGetActivityForm(w http.ResponseWriter, r *http.Request) {
	empty := db.RunDay{}
	if r.Header.Get("HX-Request") == "true" {
		pages.ActivityFormContent(empty, models.RPEOptions, app.username(r)).Render(r.Context(), w)
	} else {
		pages.ActivityForm(empty, models.RPEOptions, app.username(r)).Render(r.Context(), w)
	}
}

// handlePostActivity creates an unscheduled activity log entry
func (app *application) handlePostActivity(w http.ResponseWriter, r *http.Request) {
	userID, ok := app.getSessionUserID(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	params, err := app.parseActivityForm(r, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	entry, err := app.queries.CreateActivityLog(r.Context(), params)
	if err != nil {
		app.logger.Printf("error creating activity log: %v", err)
		http.Error(w, "failed to log activity", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/activity/%d", entry.ID), http.StatusSeeOther)
}

// handleGetActivity shows a logged activity
func (app *application) handleGetActivity(w http.ResponseWriter, r *http.Request) {
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

	entry, err := app.queries.GetActivityLogByID(r.Context(), int32(id))
	if err != nil {
		http.Error(w, "activity not found", http.StatusNotFound)
		return
	}
	if entry.UserID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		pages.ActivityDetailContent(entry, app.username(r)).Render(r.Context(), w)
	} else {
		pages.ActivityDetail(entry, app.username(r)).Render(r.Context(), w)
	}
}

// handleDeleteActivity deletes an activity log entry
func (app *application) handleDeleteActivity(w http.ResponseWriter, r *http.Request) {
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

	entry, err := app.queries.GetActivityLogByID(r.Context(), int32(id))
	if err != nil {
		http.Error(w, "activity not found", http.StatusNotFound)
		return
	}
	if entry.UserID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if err := app.queries.DeleteActivityLog(r.Context(), int32(id)); err != nil {
		app.logger.Printf("error deleting activity: %v", err)
		http.Error(w, "failed to delete activity", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// parseActivityForm is a helper that parses and validates the activity log form
// shared between the scheduled and unscheduled log handlers
func (app *application) parseActivityForm(r *http.Request, userID int32) (db.CreateActivityLogParams, error) {
	dateStr := r.FormValue("date")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return db.CreateActivityLogParams{}, fmt.Errorf("invalid date")
	}

	var distance pgtype.Float8
	if v := r.FormValue("distance"); v != "" {
		if d, err := strconv.ParseFloat(v, 64); err == nil {
			distance = pgtype.Float8{Float64: d, Valid: true}
		}
	}

	var duration pgtype.Int4
	if v := r.FormValue("duration"); v != "" {
		if d, err := strconv.Atoi(v); err == nil {
			duration = pgtype.Int4{Int32: int32(d), Valid: true}
		}
	}

	// Calculate pace if not provided but distance and duration are
	var pace pgtype.Int4
	if v := r.FormValue("pace"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			pace = pgtype.Int4{Int32: int32(p), Valid: true}
		}
	} else if distance.Valid && duration.Valid {
		calculated := models.PaceFromDistanceAndDuration(distance.Float64, int(duration.Int32))
		if calculated > 0 {
			pace = pgtype.Int4{Int32: int32(calculated), Valid: true}
		}
	}

	var rpe pgtype.Int2
	if v := r.FormValue("rpe"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i >= 1 && i <= 10 {
			rpe = pgtype.Int2{Int16: int16(i), Valid: true}
		}
	}

	var notes pgtype.Text
	if v := r.FormValue("notes"); v != "" {
		notes = pgtype.Text{String: v, Valid: true}
	}

	return db.CreateActivityLogParams{
		UserID:   userID,
		Date:     pgtype.Date{Time: date, Valid: true},
		RunType:  r.FormValue("run_type"),
		Distance: distance,
		Duration: duration,
		Pace:     pace,
		Rpe:      rpe,
		Notes:    notes,
	}, nil
}

func (app *application) handleGetEditActivity(w http.ResponseWriter, r *http.Request) {
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

	entry, err := app.queries.GetActivityLogByID(r.Context(), int32(id))
	if err != nil {
		http.Error(w, "activity not found", http.StatusNotFound)
		return
	}
	if entry.UserID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		pages.ActivityEditFormContent(entry, models.RPEOptions, app.username(r)).Render(r.Context(), w)
	} else {
		pages.ActivityEditForm(entry, models.RPEOptions, app.username(r)).Render(r.Context(), w)
	}
}

func (app *application) handlePostEditActivity(w http.ResponseWriter, r *http.Request) {
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

	entry, err := app.queries.GetActivityLogByID(r.Context(), int32(id))
	if err != nil {
		http.Error(w, "activity not found", http.StatusNotFound)
		return
	}
	if entry.UserID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Reuse parseActivityForm for the field parsing
	parsed, err := app.parseActivityForm(r, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = app.queries.UpdateActivityLog(r.Context(), db.UpdateActivityLogParams{
		ID:       int32(id),
		Distance: parsed.Distance,
		Duration: parsed.Duration,
		Pace:     parsed.Pace,
		Rpe:      parsed.Rpe,
		Notes:    parsed.Notes,
	})
	if err != nil {
		app.logger.Printf("error updating activity: %v", err)
		http.Error(w, "failed to update activity", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/activity/%d", id), http.StatusSeeOther)
}
