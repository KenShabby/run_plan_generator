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

	return r

}
