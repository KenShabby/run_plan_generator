package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/KenShabby/run_plan_generator/internal/db"
	"github.com/KenShabby/run_plan_generator/internal/models"
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
	r.Post("/plans/{id}/runs", app.handlePostPlansByIdRuns)
	r.Delete("/plans/{id}", app.handleDeletePlan)
	r.Get("/plans/{id}/export.ics", app.handleExportPlan)
	// Run builder routes
	r.Post("/plans/{id}/runs/builder", app.handleRunBuilderAddSegment)
	r.Post("/plans/{id}/runs/builder/repeat", app.handleRunBuilderAddRepeat)
	r.Post("/plans/{id}/runs/builder/reorder", app.handleRunBuilderReorder)
	r.Post("/plans/{id}/runs/builder/delete", app.handleRunBuilderDelete)
	r.Post("/plans/{id}/runs/builder/add-to-block", app.handleRunBuilderAddToBlock)
	r.Post("/plans/{id}/runs/builder/close-block", app.handleRunBuilderCloseBlock)
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
	pages.RunForm(int32(id), app.username(r)).Render(r.Context(), w)

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

	segments := parseSegmentInputs(r)

	// Check if there's a pending segment in the new_* fields
	// If effort_type is filled in, the user has entered a segment but not hit Add Segment
	if effortType := r.FormValue("new_effort_type"); effortType != "" {
		dist, _ := strconv.ParseFloat(r.FormValue("new_distance"), 64)
		dur, _ := models.ParseDuration(r.FormValue("new_duration"))
		hrMin, _ := strconv.Atoi(r.FormValue("new_hr_zone_min"))
		hrMax, _ := strconv.Atoi(r.FormValue("new_hr_zone_max"))

		// Check if we're in an open block
		openSetIndex, _ := strconv.Atoi(r.FormValue("open_set_index"))
		openSetReps, _ := strconv.Atoi(r.FormValue("open_set_reps"))

		pending := models.SegmentInput{
			Index:          len(segments),
			Description:    r.FormValue("new_description"),
			EffortType:     effortType,
			Distance:       dist,
			Duration:       dur,
			HrZoneMin:      hrMin,
			HrZoneMax:      hrMax,
			SetIndex:       openSetIndex,
			SetRepetitions: openSetReps,
		}
		segments = append(segments, pending)
		segments = reindexSegments(segments)
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

	// After creating run, create any segments
	segments = parseSegmentInputs(r)
	for _, seg := range segments {
		var dist pgtype.Float8
		var dur pgtype.Int8
		var hrMin, hrMax pgtype.Int4
		var setIdx, setReps pgtype.Int4

		if seg.Distance > 0 {
			dist = pgtype.Float8{Float64: seg.Distance, Valid: true}
		}
		if seg.Duration > 0 {
			dur = pgtype.Int8{Int64: int64(seg.Duration), Valid: true}
		}
		if seg.HrZoneMin > 0 {
			hrMin = pgtype.Int4{Int32: int32(seg.HrZoneMin), Valid: true}
		}
		if seg.HrZoneMax > 0 {
			hrMax = pgtype.Int4{Int32: int32(seg.HrZoneMax), Valid: true}
		}
		if seg.SetIndex > 0 {
			setIdx = pgtype.Int4{Int32: int32(seg.SetIndex), Valid: true}
			setReps = pgtype.Int4{Int32: int32(seg.SetRepetitions), Valid: true}
		}

		_, err := app.queries.CreateSegment(r.Context(), db.CreateSegmentParams{
			RunID:      run.ID,
			OrderIndex: int32(seg.Index),
			Description: pgtype.Text{
				String: seg.Description,
				Valid:  seg.Description != "",
			},
			EffortType:     seg.EffortType,
			Distance:       dist,
			Duration:       dur,
			Pace:           pgtype.Int8{Valid: false},
			Repetitions:    1,
			HrZoneMin:      hrMin,
			HrZoneMax:      hrMax,
			HrAbsMin:       pgtype.Int4{Valid: false},
			HrAbsMax:       pgtype.Int4{Valid: false},
			SetIndex:       setIdx,
			SetRepetitions: setReps,
		})
		if err != nil {
			app.logger.Printf("ERROR creating segment for run %d: %v", run.ID, err)
		}
	}

	http.Redirect(w, r, fmt.Sprintf("/plans/%d", planID), http.StatusSeeOther)
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

// handleRunBuilderAddSegment adds a standalone segment to the in-progress run
func (app *application) handleRunBuilderAddSegment(w http.ResponseWriter, r *http.Request) {
	planID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Parse existing segments from hidden fields
	segments := parseSegmentInputs(r)

	// Parse the new segment from the "add segment" section
	dist, _ := strconv.ParseFloat(r.FormValue("new_distance"), 64)
	dur, _ := models.ParseDuration(r.FormValue("new_duration"))
	hrMin, _ := strconv.Atoi(r.FormValue("new_hr_zone_min"))
	hrMax, _ := strconv.Atoi(r.FormValue("new_hr_zone_max"))

	newSeg := models.SegmentInput{
		Index:       len(segments),
		Description: r.FormValue("new_description"),
		EffortType:  r.FormValue("new_effort_type"),
		Distance:    dist,
		Duration:    dur,
		HrZoneMin:   hrMin,
		HrZoneMax:   hrMax,
	}
	segments = append(segments, newSeg)
	segments = reindexSegments(segments)

	// Parse run basics to re-render the full form
	runBasics := parseRunBasics(r)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<div id="run-builder-container">`)
	pages.RunFormWithBuilder(int32(planID), runBasics, segments).Render(r.Context(), w)
	fmt.Fprintf(w, `</div>`)
}

// handleRunBuilderAddRepeat adds a repeat block with N repetitions
func (app *application) handleRunBuilderAddRepeat(w http.ResponseWriter, r *http.Request) {
	planID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	segments := parseSegmentInputs(r)
	runBasics := parseRunBasics(r)

	// Find the next available set index
	maxSetIdx := 0
	for _, s := range segments {
		if s.SetIndex > maxSetIdx {
			maxSetIdx = s.SetIndex
		}
	}

	reps, _ := strconv.Atoi(r.FormValue("new_repeat_reps"))
	if reps < 2 {
		reps = 2
	}

	// Open a new block — don't add any segments yet
	runBasics.OpenSetIndex = maxSetIdx + 1
	runBasics.OpenSetReps = reps

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<div id="run-builder-container">`)
	pages.RunFormWithBuilder(int32(planID), runBasics, segments).Render(r.Context(), w)
	fmt.Fprintf(w, `</div>`)

}

// handleRunBuilderReorder moves a segment up or down
func (app *application) handleRunBuilderReorder(w http.ResponseWriter, r *http.Request) {
	planID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	segments := parseSegmentInputs(r)
	idx, _ := strconv.Atoi(r.FormValue("target_index"))
	direction := r.FormValue("direction")
	segments = moveSegment(segments, idx, direction)

	runBasics := parseRunBasics(r)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<div id="run-builder-container">`)
	pages.RunFormWithBuilder(int32(planID), runBasics, segments).Render(r.Context(), w)
	fmt.Fprintf(w, `</div>`)
}

// handleRunBuilderDelete removes a segment from the in-progress list
func (app *application) handleRunBuilderDelete(w http.ResponseWriter, r *http.Request) {
	planID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	segments := parseSegmentInputs(r)
	idx, _ := strconv.Atoi(r.FormValue("target_index"))
	segments = deleteSegment(segments, idx)

	runBasics := parseRunBasics(r)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<div id="run-builder-container">`)
	pages.RunFormWithBuilder(int32(planID), runBasics, segments).Render(r.Context(), w)
	fmt.Fprintf(w, `</div>`)
}

// handleRunBuilderAddToBlock adds a segment to the currently open repeat block
func (app *application) handleRunBuilderAddToBlock(w http.ResponseWriter, r *http.Request) {
	planID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	segments := parseSegmentInputs(r)
	runBasics := parseRunBasics(r)

	effortType := r.FormValue("new_effort_type")
	if effortType == "" {
		effortType = "distance" // sensible default
	}

	dist, _ := strconv.ParseFloat(r.FormValue("new_distance"), 64)
	dur, _ := models.ParseDuration(r.FormValue("new_duration"))
	hrMin, _ := strconv.Atoi(r.FormValue("new_hr_zone_min"))
	hrMax, _ := strconv.Atoi(r.FormValue("new_hr_zone_max"))

	newSeg := models.SegmentInput{
		Index:          len(segments),
		Description:    r.FormValue("new_description"),
		EffortType:     r.FormValue("new_effort_type"),
		Distance:       dist,
		Duration:       dur,
		HrZoneMin:      hrMin,
		HrZoneMax:      hrMax,
		SetIndex:       runBasics.OpenSetIndex,
		SetRepetitions: runBasics.OpenSetReps,
	}
	segments = append(segments, newSeg)
	segments = reindexSegments(segments)

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<div id="run-builder-container">`)
	pages.RunFormWithBuilder(int32(planID), runBasics, segments).Render(r.Context(), w)
	fmt.Fprintf(w, `</div>`)
}

// handleRunBuilderCloseBlock closes the currently open repeat block
func (app *application) handleRunBuilderCloseBlock(w http.ResponseWriter, r *http.Request) {
	planID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	segments := parseSegmentInputs(r)
	runBasics := parseRunBasics(r)

	// Check that the block has at least one segment
	hasSegments := false
	for _, s := range segments {
		if s.SetIndex == runBasics.OpenSetIndex {
			hasSegments = true
			break
		}
	}

	// Only close if there's at least one segment in the block
	if hasSegments {
		runBasics.OpenSetIndex = 0
		runBasics.OpenSetReps = 0
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<div id="run-builder-container">`)
	pages.RunFormWithBuilder(int32(planID), runBasics, segments).Render(r.Context(), w)
	fmt.Fprintf(w, `</div>`)
}
