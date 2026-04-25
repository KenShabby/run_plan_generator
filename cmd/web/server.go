package main

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KenShabby/run_plan_generator/internal/db"
	"github.com/KenShabby/run_plan_generator/internal/templates/pages"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func newServer(queries *db.Queries) http.Handler {
	r := chi.NewRouter()

	// Basic middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// root
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		pages.Index().Render(r.Context(), w)
	})

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	r.Get("/plans", func(w http.ResponseWriter, r *http.Request) {
		// Hardcode user ID for now
		plans, err := queries.ListTrainingPlansByUser(r.Context(), 1)
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
			pages.Plans(plans).Render(r.Context(), w)
		}
	})

	// handles form submission
	r.Post("/plans", func(w http.ResponseWriter, r *http.Request) {
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

		plan, err := queries.CreateTrainingPlan(r.Context(), db.CreateTrainingPlanParams{
			UserID:       1,
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
	})

	r.Get("/plans/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		plan, err := queries.GetTrainingPlan(r.Context(), int32(id))
		if err != nil {
			http.Error(w, "plan not found", http.StatusNotFound)
			return
		}

		runs, err := queries.ListRunDaysByPlan(r.Context(), int32(id))
		if err != nil {
			log.Printf("error fetching runs: %v", err)
			http.Error(w, "failed to load runs", http.StatusInternalServerError)
			return
		}

		if r.Header.Get("HX-Request") == "true" {
			pages.PlanDetailContent(plan, runs).Render(r.Context(), w)
		} else {
			pages.PlanDetail(plan, runs).Render(r.Context(), w)
		}
	})

	r.Get("/plans/{id}/runs/new", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		pages.RunForm(int32(id)).Render(r.Context(), w)
	})

	r.Get("/plans/{id}/runs/form/cancel", func(w http.ResponseWriter, r *http.Request) {
		pages.RunFormEmpty().Render(r.Context(), w)
	})

	r.Post("/plans/{id}/runs", func(w http.ResponseWriter, r *http.Request) {
		planID, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
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

		run, err := queries.CreateRunDay(r.Context(), db.CreateRunDayParams{
			PlanID:        int32(planID),
			Date:          pgtype.Date{Time: date, Valid: true},
			RunType:       r.FormValue("run_type"),
			TotalDistance: dist,
			TotalDuration: pgtype.Int8{Valid: false},
			Notes:         pgtype.Text{String: r.FormValue("notes"), Valid: r.FormValue("notes") != ""},
		})
		if err != nil {
			log.Printf("error creating run: %v", err)
			http.Error(w, "failed to create run", http.StatusInternalServerError)
			return
		}

		pages.RunCard(run).Render(r.Context(), w)
	})

	r.Delete("/plans/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		if err := queries.DeleteTrainingPlan(r.Context(), int32(id)); err != nil {
			log.Printf("error deleting plan: %v", err)
			http.Error(w, "failed to delete plan", http.StatusInternalServerError)
			return
		}

		// return empty 200 - HTMX will swap outerHTML with nothing, removing the card
		w.WriteHeader(http.StatusOK)
	})

	// clears the form container on cancel
	r.Get("/plans/form/cancel", func(w http.ResponseWriter, r *http.Request) {
		pages.PlanFormEmpty().Render(r.Context(), w)
	})

	// serves the form fragment
	r.Get("/plans/new", func(w http.ResponseWriter, r *http.Request) {
		pages.PlanForm().Render(r.Context(), w)
	})

	r.Delete("/runs/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		if err := queries.DeleteRunDay(r.Context(), int32(id)); err != nil {
			log.Printf("error deleting run: %v", err)
			http.Error(w, "failed to delete run", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	r.Get("/templates", func(w http.ResponseWriter, r *http.Request) {
		plans, err := queries.ListTemplatePlansWithCounts(r.Context())
		if err != nil {
			log.Printf("error fetching templates: %v", err)
			http.Error(w, "failed to load templates", http.StatusInternalServerError)
			return
		}
		if r.Header.Get("HX-Request") == "true" {
			pages.TemplatesContent(plans).Render(r.Context(), w)
		} else {
			pages.Templates(plans).Render(r.Context(), w)
		}
	})

	r.Get("/templates/{id}/select", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		tmpl, err := queries.GetTemplatePlan(r.Context(), int32(id))
		if err != nil {
			http.Error(w, "template not found", http.StatusNotFound)
			return
		}
		pages.TemplateSelectForm(tmpl).Render(r.Context(), w)
	})

	r.Get("/templates/form/cancel", func(w http.ResponseWriter, r *http.Request) {
		pages.TemplateFormEmpty().Render(r.Context(), w)
	})

	r.Post("/templates/{id}/instantiate", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		raceDate, err := time.Parse("2006-01-02", r.FormValue("race_date"))
		if err != nil {
			http.Error(w, "invalid race date", http.StatusBadRequest)
			return
		}

		tmpl, err := queries.GetTemplatePlan(r.Context(), int32(id))
		if err != nil {
			http.Error(w, "template not found", http.StatusNotFound)
			return
		}

		startDate := raceDate.AddDate(0, 0, -(int(tmpl.TotalWeeks) * 7))

		plan, err := queries.CreateTrainingPlan(r.Context(), db.CreateTrainingPlanParams{
			UserID:       1,
			Name:         r.FormValue("name"),
			Description:  pgtype.Text{String: tmpl.Description.String, Valid: tmpl.Description.Valid},
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
		templateRuns, err := queries.ListTemplateRunDaysByPlan(r.Context(), tmpl.ID)
		if err != nil {
			log.Printf("error fetching template runs: %v", err)
			http.Error(w, "failed to load template runs", http.StatusInternalServerError)
			return
		}

		for _, tr := range templateRuns {
			runDate := startDate.AddDate(0, 0, int(tr.DayOffset))
			_, err := queries.CreateRunDay(r.Context(), db.CreateRunDayParams{
				PlanID:        plan.ID,
				Date:          pgtype.Date{Time: runDate, Valid: true},
				RunType:       tr.RunType,
				TotalDistance: tr.Distance,
				TotalDuration: pgtype.Int8{Valid: false},
				Notes:         tr.Notes,
			})
			if err != nil {
				log.Printf("error creating run day: %v", err)
				http.Error(w, "failed to create run days", http.StatusInternalServerError)
				return
			}
		}

		// redirect to the new plan
		plans, _ := queries.ListPlansByUser(r.Context(), 1)
		pages.PlansContent(plans).Render(r.Context(), w)
	})

	return r

}
