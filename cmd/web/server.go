package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/KenShabby/run_plan_generator/internal/db"
	"github.com/KenShabby/run_plan_generator/internal/templates/pages"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func newServer(app *application) http.Handler {
	r := chi.NewRouter()
	// Basic middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(app.loadUser)

	// Public routes, no auth required
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		username := app.username(r)
		pages.Index(username).Render(r.Context(), w)
	})

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	r.Get("/login", func(w http.ResponseWriter, r *http.Request) {
		username := app.username(r)
		pages.Login("", username).Render(r.Context(), w)
	})

	r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
		username := app.username(r)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		email := strings.TrimSpace(r.FormValue("email"))
		password := r.FormValue("password")

		user, err := app.queries.GetUserByEmail(r.Context(), email)
		if err != nil {
			pages.Login("Invalid email or password.", username).Render(r.Context(), w)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
			pages.Login("Invalid email or password.", username).Render(r.Context(), w)
			return
		}

		if err := app.setSessionUserID(w, r, user.ID); err != nil {
			log.Printf("session error: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/plans", http.StatusSeeOther)
	})

	// GET /register — serve the form
	r.Get("/register", func(w http.ResponseWriter, r *http.Request) {
		username := app.username(r)
		pages.Register("", username).Render(r.Context(), w)
	})

	// POST /register — handle submission
	r.Post("/register", func(w http.ResponseWriter, r *http.Request) {
		navUsername := app.username(r)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		newUsername := strings.TrimSpace(r.FormValue("username"))
		email := strings.TrimSpace(r.FormValue("email"))
		password := r.FormValue("password")
		confirm := r.FormValue("confirm_password")

		if newUsername == "" || email == "" || password == "" {
			pages.Register("All fields are required.", navUsername).Render(r.Context(), w)
			return
		}
		if password != confirm {
			pages.Register("Passwords do not match.", navUsername).Render(r.Context(), w)
			return
		}
		if len(password) < 8 {
			pages.Register("Password must be at least 8 characters.", navUsername).Render(r.Context(), w)
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("bcrypt error: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		user, err := app.queries.CreateUser(r.Context(), db.CreateUserParams{
			Username:     newUsername,
			Email:        email,
			PasswordHash: string(hash),
		})
		if err != nil {
			if strings.Contains(err.Error(), "23505") {
				pages.Register("That username or email is already taken.", navUsername).Render(r.Context(), w)
				return
			}
			log.Printf("CreateUser error: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if err := app.setSessionUserID(w, r, user.ID); err != nil {
			log.Printf("session error: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/plans", http.StatusSeeOther)
	})

	// Protected groups - auth required
	r.Group(func(r chi.Router) {
		r.Use(app.requireAuth)
		r.Post("/logout", func(w http.ResponseWriter, r *http.Request) {
			app.clearSession(w, r)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
		})

		r.Get("/account", func(w http.ResponseWriter, r *http.Request) {
			user, ok := userFromContext(r.Context())
			if !ok {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			pages.Account(user, "", "", "", app.username(r)).Render(r.Context(), w)
		})

		r.Post("/account/username", func(w http.ResponseWriter, r *http.Request) {
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
		})

		r.Post("/account/email", func(w http.ResponseWriter, r *http.Request) {
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
		})

		r.Post("/account/password", func(w http.ResponseWriter, r *http.Request) {
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
		})

		r.Post("/account/delete", func(w http.ResponseWriter, r *http.Request) {
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
		})

		// serves the form fragment
		r.Get("/plans/new", func(w http.ResponseWriter, r *http.Request) {
			pages.PlanForm().Render(r.Context(), w)
		})

		r.Get("/plans/form/cancel", func(w http.ResponseWriter, r *http.Request) {
			pages.PlanFormEmpty().Render(r.Context(), w)
		})

		r.Get("/plans", func(w http.ResponseWriter, r *http.Request) {
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
		})

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
		})

		r.Get("/plans/{id}", func(w http.ResponseWriter, r *http.Request) {
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
		})

		r.Get("/plans/{id}/runs/new", func(w http.ResponseWriter, r *http.Request) {
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
			})
			if err != nil {
				log.Printf("error creating run: %v", err)
				http.Error(w, "failed to create run", http.StatusInternalServerError)
				return
			}

			pages.RunCard(run).Render(r.Context(), w)
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
				pages.RunDetailContent(runDay, segments).Render(r.Context(), w)
			} else {
				pages.RunDetail(runDay, segments, username).Render(r.Context(), w)
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

			for _, tr := range templateRuns {
				runDate := startDate.AddDate(0, 0, int(tr.DayOffset))
				run, err := app.queries.CreateRunDay(r.Context(), db.CreateRunDayParams{
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
