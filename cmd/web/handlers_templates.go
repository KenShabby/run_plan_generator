package main

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/KenShabby/run_plan_generator/internal/db"
	"github.com/KenShabby/run_plan_generator/internal/templates/pages"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (app *application) registerTemplateRoutes(r chi.Router) {
	r.Get("/templates", app.handleGetTemplates)
	r.Get("/templates/{id}/select", app.handleGetTemplateById)
	r.Post("/templates/{id}/instantiate", app.handleTemplateInstantiate)
}

func (app *application) handleGetTemplates(w http.ResponseWriter, r *http.Request) {
	username := app.username(r)
	plans, err := app.queries.ListTemplatePlansWithCounts(r.Context())
	if err != nil {
		log.Printf("error fetching templates: %v", err)
		http.Error(w, "failed to load templates", http.StatusInternalServerError)
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		pages.TemplatesContent(plans).Render(r.Context(), w)
	} else {
		pages.Templates(plans, username).Render(r.Context(), w)
	}
}

func (app *application) handleGetTemplateById(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	tmpl, err := app.queries.GetTemplatePlan(r.Context(), int32(id))
	if err != nil {
		http.Error(w, "template not found", http.StatusNotFound)
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		pages.TemplateSelectContent(tmpl, app.username(r)).Render(r.Context(), w)
	} else {
		pages.TemplateSelectPage(tmpl, app.username(r)).Render(r.Context(), w)
	}
}

func (app *application) handleTemplateInstantiate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	raceDate, err := time.ParseInLocation("2006-01-02", r.FormValue("race_date"), time.Local)
	if err != nil {
		http.Error(w, "invalid race date", http.StatusBadRequest)
		return
	}

	tmpl, err := app.queries.GetTemplatePlan(r.Context(), int32(id))
	if err != nil {
		http.Error(w, "template not found", http.StatusNotFound)
		return
	}

	startDate := raceDate.AddDate(0, 0, -(int(tmpl.TotalWeeks) * 7))
	// Normalise to Monday of that week
	weekday := int(startDate.Weekday())
	if weekday == 0 {
		// Sunday - advance to next Monday rather than going back 6 days
		startDate = startDate.AddDate(0, 0, 1)
	} else if weekday != 1 {
		// Not Monday - go back to Monday of this week
		startDate = startDate.AddDate(0, 0, -(weekday - 1))
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
		PlanType:     tmpl.PlanType,
		DistanceUnit: tmpl.DistanceUnit,
		StartDate:    pgtype.Date{Time: startDate, Valid: true},
		EndDate:      pgtype.Date{Time: raceDate, Valid: true},
		TemplateID:   pgtype.Int4{Int32: tmpl.ID, Valid: true},
	})
	if err != nil {
		log.Printf("error creating plan: %v", err)
		http.Error(w, "failed to create plan", http.StatusInternalServerError)
		return
	}

	// populate run days from template
	templateRuns, err := app.queries.ListTemplateRunDaysByPlan(r.Context(), tmpl.ID)
	if err != nil {
		log.Printf("error fetching template runs: %v", err)
		http.Error(w, "failed to load template runs", http.StatusInternalServerError)
		return
	}

	for i, tr := range templateRuns {
		runDate := startDate.AddDate(0, 0, int(tr.DayOffset))
		isGoalRace := tr.RunType == "race" && i == len(templateRuns)-1

		run, err := app.queries.CreateRunDay(r.Context(), db.CreateRunDayParams{
			PlanID:        plan.ID,
			Date:          pgtype.Date{Time: runDate, Valid: true},
			RunType:       tr.RunType,
			TotalDistance: tr.Distance,
			TotalDuration: pgtype.Int8{Valid: false},
			Notes:         tr.Notes,
			IsGoalRace:    isGoalRace,
		})
		if err != nil {
			log.Printf("error creating run day: %v", err)
			http.Error(w, "failed to create run days", http.StatusInternalServerError)
			return
		}

		// Copy segments from template
		templateSegs, err := app.queries.ListTemplateSegmentsByRun(r.Context(), tr.ID)
		if err != nil {
			log.Printf("error fetching template segments: %v", err)
			http.Error(w, "failed to load template segments", http.StatusInternalServerError)
			return
		}

		for _, ts := range templateSegs {
			_, err := app.queries.CreateSegment(r.Context(), db.CreateSegmentParams{
				RunID:          run.ID,
				OrderIndex:     ts.OrderIndex,
				Description:    ts.Description,
				EffortType:     ts.EffortType,
				Distance:       ts.Distance,
				Duration:       ts.Duration,
				Pace:           ts.Pace,
				Repetitions:    ts.Repetitions,
				HrZoneMin:      ts.HrZoneMin,
				HrZoneMax:      ts.HrZoneMax,
				HrAbsMin:       pgtype.Int4{Valid: false},
				HrAbsMax:       pgtype.Int4{Valid: false},
				SetIndex:       ts.SetIndex,
				SetRepetitions: ts.SetRepetitions,
			})
			if err != nil {
				log.Printf("error creating segment: %v", err)
				http.Error(w, "failed to create segments", http.StatusInternalServerError)
				return
			}
		}
	}

	userID, ok = app.getSessionUserID(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	plans, _ := app.queries.ListTrainingPlansByUser(r.Context(), userID)
	pages.PlansContent(plans).Render(r.Context(), w)
}
