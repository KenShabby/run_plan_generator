package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/KenShabby/run_plan_generator/internal/db"
	"github.com/KenShabby/run_plan_generator/internal/models"
	"github.com/KenShabby/run_plan_generator/internal/templates/pages"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// handleGetEditRun shows the edit form pre-populated with existing run data
func (app *application) handleGetEditRun(w http.ResponseWriter, r *http.Request) {
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

	// Load existing segments
	dbSegs, err := app.queries.ListSegmentsByRun(r.Context(), int32(id))
	if err != nil {
		app.logger.Printf("error fetching segments for run %d: %v", id, err)
	}

	// Convert db segments to SegmentInput
	segments := dbSegmentsToInputs(dbSegs)

	// Build RunBasics from existing run
	basics := models.RunBasics{
		Date:          run.Date.Time.Format("2006-01-02"),
		RunType:       run.RunType,
		TotalDistance: "",
		Notes:         "",
	}
	if run.TotalDistance.Valid {
		basics.TotalDistance = fmt.Sprintf("%.2f", run.TotalDistance.Float64)
	}
	if run.Notes.Valid {
		basics.Notes = run.Notes.String
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Push-Url", fmt.Sprintf("/runs/%d/edit", id))
		pages.RunEditFormContent(int32(id), int32(run.PlanID), basics, segments, app.username(r)).Render(r.Context(), w)
	} else {
		pages.RunEditForm(int32(id), int32(run.PlanID), basics, segments, app.username(r)).Render(r.Context(), w)
	}
}

// handlePostEditRun saves the edited run and replaces all segments
func (app *application) handlePostEditRun(w http.ResponseWriter, r *http.Request) {
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

	// Parse run basics
	date, err := parseDate(r.FormValue("date"))
	if err != nil {
		http.Error(w, "invalid date", http.StatusBadRequest)
		return
	}

	var dist pgtype.Float8
	if v := r.FormValue("total_distance"); v != "" {
		if d, err := strconv.ParseFloat(v, 64); err == nil {
			dist = pgtype.Float8{Float64: d, Valid: true}
		}
	}

	var notes pgtype.Text
	if v := r.FormValue("notes"); v != "" {
		notes = pgtype.Text{String: v, Valid: true}
	}

	// Update the run day
	_, err = app.queries.UpdateRunDay(r.Context(), db.UpdateRunDayParams{
		ID:            int32(id),
		Date:          pgtype.Date{Time: date, Valid: true},
		RunType:       r.FormValue("run_type"),
		TotalDistance: dist,
		Notes:         notes,
	})
	if err != nil {
		app.logger.Printf("error updating run %d: %v", id, err)
		http.Error(w, "failed to update run", http.StatusInternalServerError)
		return
	}

	// Delete all existing segments and recreate
	if err := app.queries.DeleteSegmentsByRun(r.Context(), int32(id)); err != nil {
		app.logger.Printf("error deleting segments for run %d: %v", id, err)
	}

	// Parse and create new segments
	segments := parseSegmentInputs(r)

	// Flush any pending segment
	if effortType := r.FormValue("new_effort_type"); effortType != "" {
		dist, _ := strconv.ParseFloat(r.FormValue("new_distance"), 64)
		dur, _ := models.ParseDuration(r.FormValue("new_duration"))
		hrMin, _ := strconv.Atoi(r.FormValue("new_hr_zone_min"))
		hrMax, _ := strconv.Atoi(r.FormValue("new_hr_zone_max"))
		openSetIndex, _ := strconv.Atoi(r.FormValue("open_set_index"))
		openSetReps, _ := strconv.Atoi(r.FormValue("open_set_reps"))

		segments = append(segments, models.SegmentInput{
			Index:          len(segments),
			Description:    r.FormValue("new_description"),
			EffortType:     effortType,
			Distance:       dist,
			Duration:       dur,
			HrZoneMin:      hrMin,
			HrZoneMax:      hrMax,
			SetIndex:       openSetIndex,
			SetRepetitions: openSetReps,
		})
		segments = reindexSegments(segments)
	}

	for _, seg := range segments {
		var segDist pgtype.Float8
		var segDur pgtype.Int8
		var hrMin, hrMax pgtype.Int4
		var setIdx, setReps pgtype.Int4

		if seg.Distance > 0 {
			segDist = pgtype.Float8{Float64: seg.Distance, Valid: true}
		}
		if seg.Duration > 0 {
			segDur = pgtype.Int8{Int64: int64(seg.Duration), Valid: true}
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
			RunID:      int32(id),
			OrderIndex: int32(seg.Index),
			Description: pgtype.Text{
				String: seg.Description,
				Valid:  seg.Description != "",
			},
			EffortType:     seg.EffortType,
			Distance:       segDist,
			Duration:       segDur,
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
			app.logger.Printf("error creating segment for run %d: %v", id, err)
		}
	}

	updatedRun, err := app.queries.GetRunDayWithPlanOwner(r.Context(), int32(id))
	if err != nil {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}

	updatedSegs, _ := app.queries.ListSegmentsByRun(r.Context(), int32(id))
	hrzones, _ := app.queries.GetHRZonesByUser(r.Context(), userID)
	zoneMap := make(map[int32]db.HrZone)
	for _, z := range hrzones {
		zoneMap[z.ZoneNumber] = z
	}
	runDay := db.RunDay{
		ID:            updatedRun.ID,
		PlanID:        updatedRun.PlanID,
		Date:          updatedRun.Date,
		RunType:       updatedRun.RunType,
		TotalDistance: updatedRun.TotalDistance,
		TotalDuration: updatedRun.TotalDuration,
		Completed:     updatedRun.Completed,
		Notes:         updatedRun.Notes,
		CreatedAt:     updatedRun.CreatedAt,
	}
	pages.RunDetailContent(runDay, updatedSegs, zoneMap, int(updatedRun.PlanID)).Render(r.Context(), w)
}

// handleRunEditBuilderAddSegment adds a standalone segment in the edit flow
func (app *application) handleRunEditBuilderAddSegment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	segments := parseSegmentInputs(r)
	dist, _ := strconv.ParseFloat(r.FormValue("new_distance"), 64)
	dur, _ := models.ParseDuration(r.FormValue("new_duration"))
	hrMin, _ := strconv.Atoi(r.FormValue("new_hr_zone_min"))
	hrMax, _ := strconv.Atoi(r.FormValue("new_hr_zone_max"))

	segments = append(segments, models.SegmentInput{
		Index:       len(segments),
		Description: r.FormValue("new_description"),
		EffortType:  r.FormValue("new_effort_type"),
		Distance:    dist,
		Duration:    dur,
		HrZoneMin:   hrMin,
		HrZoneMax:   hrMax,
	})
	segments = reindexSegments(segments)

	basics := parseRunBasics(r)
	pages.RunEditFormContent(int32(id), int32(0), basics, segments, "").Render(r.Context(), w)
}

// handleRunEditBuilderAddRepeat opens a new repeat block in the edit flow
func (app *application) handleRunEditBuilderAddRepeat(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	segments := parseSegmentInputs(r)
	basics := parseRunBasics(r)

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

	basics.OpenSetIndex = maxSetIdx + 1
	basics.OpenSetReps = reps

	pages.RunEditFormContent(int32(id), int32(0), basics, segments, "").Render(r.Context(), w)
}

// handleRunEditBuilderAddToBlock adds a segment to the open repeat block
func (app *application) handleRunEditBuilderAddToBlock(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	segments := parseSegmentInputs(r)
	basics := parseRunBasics(r)

	effortType := r.FormValue("new_effort_type")
	if effortType == "" {
		effortType = "distance" // sensible default
	}

	dist, _ := strconv.ParseFloat(r.FormValue("new_distance"), 64)
	dur, _ := models.ParseDuration(r.FormValue("new_duration"))
	hrMin, _ := strconv.Atoi(r.FormValue("new_hr_zone_min"))
	hrMax, _ := strconv.Atoi(r.FormValue("new_hr_zone_max"))

	segments = append(segments, models.SegmentInput{
		Index:          len(segments),
		Description:    r.FormValue("new_description"),
		EffortType:     r.FormValue("new_effort_type"),
		Distance:       dist,
		Duration:       dur,
		HrZoneMin:      hrMin,
		HrZoneMax:      hrMax,
		SetIndex:       basics.OpenSetIndex,
		SetRepetitions: basics.OpenSetReps,
	})
	segments = reindexSegments(segments)

	pages.RunEditFormContent(int32(id), int32(0), basics, segments, "").Render(r.Context(), w)
}

// handleRunEditBuilderCloseBlock closes the open repeat block
func (app *application) handleRunEditBuilderCloseBlock(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	segments := parseSegmentInputs(r)
	basics := parseRunBasics(r)

	hasSegments := false
	for _, s := range segments {
		if s.SetIndex == basics.OpenSetIndex {
			hasSegments = true
			break
		}
	}
	if hasSegments {
		basics.OpenSetIndex = 0
		basics.OpenSetReps = 0
	}

	pages.RunEditFormContent(int32(id), int32(0), basics, segments, "").Render(r.Context(), w)
}

// handleRunEditBuilderReorder moves a segment up or down in the edit flow
func (app *application) handleRunEditBuilderReorder(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
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

	basics := parseRunBasics(r)
	pages.RunEditFormContent(int32(id), int32(0), basics, segments, "").Render(r.Context(), w)
}

// handleRunEditBuilderDelete removes a segment in the edit flow
func (app *application) handleRunEditBuilderDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
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

	basics := parseRunBasics(r)
	pages.RunEditFormContent(int32(id), int32(0), basics, segments, "").Render(r.Context(), w)
}

// dbSegmentsToInputs converts db.Segment slice to SegmentInput slice
func dbSegmentsToInputs(segs []db.Segment) []models.SegmentInput {
	inputs := make([]models.SegmentInput, len(segs))
	for i, s := range segs {
		var dist float64
		var dur int
		var hrMin, hrMax int
		var setIdx, setReps int

		if s.Distance.Valid {
			dist = s.Distance.Float64
		}
		if s.Duration.Valid {
			dur = int(s.Duration.Int64)
		}
		if s.HrZoneMin.Valid {
			hrMin = int(s.HrZoneMin.Int32)
		}
		if s.HrZoneMax.Valid {
			hrMax = int(s.HrZoneMax.Int32)
		}
		if s.SetIndex.Valid {
			setIdx = int(s.SetIndex.Int32)
		}
		if s.SetRepetitions.Valid {
			setReps = int(s.SetRepetitions.Int32)
		}

		desc := ""
		if s.Description.Valid {
			desc = s.Description.String
		}

		inputs[i] = models.SegmentInput{
			Index:          i,
			Description:    desc,
			EffortType:     s.EffortType,
			Distance:       dist,
			Duration:       dur,
			HrZoneMin:      hrMin,
			HrZoneMax:      hrMax,
			SetIndex:       setIdx,
			SetRepetitions: setReps,
		}
	}
	return inputs
}
