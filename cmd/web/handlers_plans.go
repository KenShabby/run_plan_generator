package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/KenShabby/run_plan_generator/internal/db"
	"github.com/KenShabby/run_plan_generator/internal/templates/pages"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (app *application) registerPlanRoutes(r chi.Router) {
	r.Get("/plans/new", app.handleGetPlansNew)
	r.Get("/plans/form/cancel", app.handleGetPlansFormCancel)
	r.Get("/plans", app.handleGetPlans)
	r.Post("/plans", app.handlePostPlans)
	r.Get("/plans/{id}", app.handleGetPlansById)
	r.Get("/plans/{id}/runs/new", app.handleGetPlansByIdRunsNew)
	r.Get("/plans/{id}/runs/form/cancel", app.handleGetPlansByIdRunsFormCancel)
	r.Post("/plans/{id}/runs", app.handlePostPlansByIdRuns)
	r.Delete("/plans/{id}", app.handleDeletePlan)
	r.Get("/plans/{id}/export.ics", app.handleExportPlan)
}

func (app *application) handleGetPlansNew(w http.ResponseWriter, r *http.Request) {
	pages.PlanForm().Render(r.Context(), w)
}

func (app *application) handleGetPlansFormCancel(w http.ResponseWriter, r *http.Request) {
	pages.PlanFormEmpty().Render(r.Context(), w)
}

func (app *application) handleGetPlans(w http.ResponseWriter, r *http.Request) {
	username := app.username(r)
	userID, ok := app.getSessionUserID(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	plans, err := app.queries.ListTrainingPlansByUser(r.Context(), userID)
	if err != nil {
		log.Printf("error fetching plans: %v", err)
		http.Error(w, "failed to load plans", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		// HTMX request - return just the inner content
		pages.PlansContent(plans).Render(r.Context(), w)
	} else {
		// Full page load - return the whole layout
		pages.Plans(plans, username).Render(r.Context(), w)
	}
}

func (app *application) handlePostPlans(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	startDate, err := time.Parse("2006-01-02", r.FormValue("start_date"))
	if err != nil {
		http.Error(w, "invalid start date", http.StatusBadRequest)
		return
	}
	endDate, err := time.Parse("2006-01-02", r.FormValue("end_date"))
	if err != nil {
		http.Error(w, "invalid end date", http.StatusBadRequest)
		return
	}

	userID, ok := app.getSessionUserID(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	plan, err := app.queries.CreateTrainingPlan(r.Context(), db.CreateTrainingPlanParams{
		UserID:       userID,
		Name:         r.FormValue("name"),
		Description:  pgtype.Text{String: r.FormValue("description"), Valid: r.FormValue("description") != ""},
		PlanType:     r.FormValue("plan_type"),
		DistanceUnit: r.FormValue("distance_unit"),
		StartDate:    pgtype.Date{Time: startDate, Valid: true},
		EndDate:      pgtype.Date{Time: endDate, Valid: true},
	})
	if err != nil {
		log.Printf("error creating plan: %v", err)
		http.Error(w, "failed to create plan", http.StatusInternalServerError)
		return
	}

	// return just the new card to swap into #plans-list
	pages.PlanCard(plan).Render(r.Context(), w)

}

func (app *application) handleGetPlansById(w http.ResponseWriter, r *http.Request) {
	username := app.username(r)
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	plan, err := app.queries.GetTrainingPlan(r.Context(), int32(id))
	if err != nil {
		http.Error(w, "plan not found", http.StatusNotFound)
		return
	}
	// Check ownership
	userID, ok := app.getSessionUserID(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if plan.UserID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	runs, err := app.queries.ListRunDaysByPlan(r.Context(), int32(id))
	if err != nil {
		log.Printf("error fetching runs: %v", err)
		http.Error(w, "failed to load runs", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		pages.PlanDetailContent(plan, runs).Render(r.Context(), w)
	} else {
		pages.PlanDetail(plan, runs, username).Render(r.Context(), w)
	}
}

func (app *application) handleGetPlansByIdRunsNew(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	plan, err := app.queries.GetTrainingPlan(r.Context(), int32(id))
	if err != nil {
		http.Error(w, "plan not found", http.StatusNotFound)
		return
	}
	userID, ok := app.getSessionUserID(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if plan.UserID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	pages.RunForm(int32(id)).Render(r.Context(), w)

}

func (app *application) handleGetPlansByIdRunsFormCancel(w http.ResponseWriter, r *http.Request) {
	pages.RunFormEmpty().Render(r.Context(), w)
}

func (app *application) handlePostPlansByIdRuns(w http.ResponseWriter, r *http.Request) {
	planID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	plan, err := app.queries.GetTrainingPlan(r.Context(), int32(planID))
	if err != nil {
		http.Error(w, "plan not found", http.StatusNotFound)
		return
	}
	userID, ok := app.getSessionUserID(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if plan.UserID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	date, err := time.Parse("2006-01-02", r.FormValue("date"))
	if err != nil {
		http.Error(w, "invalid date", http.StatusBadRequest)
		return
	}

	distStr := r.FormValue("total_distance")
	var dist pgtype.Float8
	if distStr != "" {
		if d, err := strconv.ParseFloat(distStr, 64); err == nil {
			dist = pgtype.Float8{Float64: d, Valid: true}
		}
	}

	run, err := app.queries.CreateRunDay(r.Context(), db.CreateRunDayParams{
		PlanID:        int32(planID),
		Date:          pgtype.Date{Time: date, Valid: true},
		RunType:       r.FormValue("run_type"),
		TotalDistance: dist,
		TotalDuration: pgtype.Int8{Valid: false},
		Notes:         pgtype.Text{String: r.FormValue("notes"), Valid: r.FormValue("notes") != ""},
		IsGoalRace:    false,
	})
	if err != nil {
		log.Printf("error creating run: %v", err)
		http.Error(w, "failed to create run", http.StatusInternalServerError)
		return
	}

	pages.RunCard(run).Render(r.Context(), w)
}

func (app *application) handleDeletePlan(w http.ResponseWriter, r *http.Request) {
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

	if err := app.queries.DeleteTrainingPlanIfOwner(r.Context(), db.DeleteTrainingPlanIfOwnerParams{
		ID:     int32(id),
		UserID: userID,
	}); err != nil {
		log.Printf("error deleting plan: %v", err)
		http.Error(w, "failed to delete plan", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (app *application) handleExportPlan(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	plan, err := app.queries.GetTrainingPlan(r.Context(), int32(id))
	if err != nil {
		http.Error(w, "plan not found", http.StatusNotFound)
		return
	}

	// Check ownership
	userID, ok := app.getSessionUserID(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if plan.UserID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	runs, err := app.queries.ListRunDaysByPlan(r.Context(), int32(id))
	if err != nil {
		log.Printf("error fetching runs: %v", err)
		http.Error(w, "failed to load runs", http.StatusInternalServerError)
		return
	}

	// fetch segments for each run
	segmentsByRun := make(map[int32][]db.Segment)
	for _, run := range runs {
		segs, err := app.queries.ListSegmentsByRun(r.Context(), run.ID)
		if err != nil {
			log.Printf("error fetching segments for run %d: %v", run.ID, err)
			continue
		}
		segmentsByRun[run.ID] = segs
	}

	ics := buildICS(plan, runs, segmentsByRun)

	filename := fmt.Sprintf("%s.ics", strings.ReplaceAll(plan.Name, " ", "_"))
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Write([]byte(ics))
}
