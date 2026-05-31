package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/KenShabby/run_plan_generator/internal/models"
)

// parseSegmentInputs reads all seg[N][field] values from a parsed form
// and returns them as an ordered slice of SegmentInput.
// Caller must have already called r.ParseForm().
func parseSegmentInputs(r *http.Request) []models.SegmentInput {
	var segments []models.SegmentInput
	for i := 0; ; i++ {
		prefix := fmt.Sprintf("seg[%d]", i)

		// Check if this index exists at all by looking for description OR effort_type
		description := r.FormValue(prefix + "[description]")
		effortType := r.FormValue(prefix + "[effort_type]")
		setIndex := r.FormValue(prefix + "[set_index]")

		// If all three are empty and there's no set_index, we're done
		if description == "" && effortType == "" && setIndex == "" {
			break
		}

		// Default effort_type if missing
		if effortType == "" {
			effortType = "distance"
		}

		dist, _ := strconv.ParseFloat(r.FormValue(prefix+"[distance]"), 64)
		distanceUnit := r.FormValue(prefix + "[distance_unit]")
		if distanceUnit == "" {
			distanceUnit = "miles"
		}
		dur, _ := models.ParseDuration(r.FormValue(prefix + "[duration]"))
		hrMin, _ := strconv.Atoi(r.FormValue(prefix + "[hr_zone_min]"))
		hrMax, _ := strconv.Atoi(r.FormValue(prefix + "[hr_zone_max]"))
		setIdx, _ := strconv.Atoi(r.FormValue(prefix + "[set_index]"))
		setReps, _ := strconv.Atoi(r.FormValue(prefix + "[set_repetitions]"))

		segments = append(segments, models.SegmentInput{
			Index:          i,
			Description:    description,
			EffortType:     effortType,
			Distance:       dist,
			DistanceUnit:   distanceUnit,
			Duration:       dur,
			HrZoneMin:      hrMin,
			HrZoneMax:      hrMax,
			SetIndex:       setIdx,
			SetRepetitions: setReps,
		})
	}
	return segments
}

// reindexSegments reassigns Index fields 0..N-1 after an insert/delete/reorder
func reindexSegments(segments []models.SegmentInput) []models.SegmentInput {
	for i := range segments {
		segments[i].Index = i
	}
	return segments
}

// moveSegment moves the segment at fromIdx in the given direction ("up" or "down")
func moveSegment(segments []models.SegmentInput, fromIdx int, direction string) []models.SegmentInput {
	if direction == "up" && fromIdx > 0 {
		segments[fromIdx], segments[fromIdx-1] = segments[fromIdx-1], segments[fromIdx]
	} else if direction == "down" && fromIdx < len(segments)-1 {
		segments[fromIdx], segments[fromIdx+1] = segments[fromIdx+1], segments[fromIdx]
	}
	return reindexSegments(segments)
}

// deleteSegment removes the segment at the given index
func deleteSegment(segments []models.SegmentInput, idx int) []models.SegmentInput {
	if idx < 0 || idx >= len(segments) {
		return segments
	}
	segments = append(segments[:idx], segments[idx+1:]...)
	return reindexSegments(segments)
}

// RunBasics holds the top-level run fields during the builder flow
type RunBasics struct {
	Date          string
	RunType       string
	TotalDistance string
	Notes         string
}

func parseRunBasics(r *http.Request) models.RunBasics {
	openSetIndex, _ := strconv.Atoi(r.FormValue("open_set_index"))
	openSetReps, _ := strconv.Atoi(r.FormValue("open_set_reps"))
	return models.RunBasics{
		Date:          r.FormValue("date"),
		RunType:       r.FormValue("run_type"),
		TotalDistance: r.FormValue("total_distance"),
		Notes:         r.FormValue("notes"),
		OpenSetIndex:  openSetIndex,
		OpenSetReps:   openSetReps,
	}
}

func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}
