package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KenShabby/run_plan_generator/internal/db"
	"github.com/KenShabby/run_plan_generator/internal/templates/pages"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func newServer(app *application) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger, middleware.Recoverer, app.loadUser)

	// Public routes, no auth required
	r.Get("/", app.handleHome)
	r.Get("/health", app.handleHealth)
	app.registerAuthRoutes(r)

	// Protected groups - auth required
	r.Group(func(r chi.Router) {
		r.Use(app.requireAuth)
		app.registerAccountRoutes(r)
		app.registerPlanRoutes(r)
		app.registerRunRoutes(r)
		app.registerTemplateRoutes(r)

		r.Post("/logout", func(w http.ResponseWriter, r *http.Request) {
			app.clearSession(w, r)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
		})

		r.Delete("/plans/{id}", func(w http.ResponseWriter, r *http.Request) {
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
		})

		r.Get("/plans/{id}/export.ics", func(w http.ResponseWriter, r *http.Request) {
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
		})

		r.Delete("/runs/{id}", func(w http.ResponseWriter, r *http.Request) {
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
		})

		r.Get("/runs/{id}", func(w http.ResponseWriter, r *http.Request) {
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
		})

		r.Get("/templates", func(w http.ResponseWriter, r *http.Request) {
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
		})

		r.Get("/templates/{id}/select", func(w http.ResponseWriter, r *http.Request) {
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
		})
	})

	return r

}
