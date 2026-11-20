package main

import (
	"context"
	"time"

	"github.com/KenShabby/run_plan_generator/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

// copyTemplateRuns writes every run and segment of tmpl into planID.
// day_offset 0 lands on `start`.
func (app *application) copyTemplateRuns(
	ctx context.Context,
	tmpl db.TemplatePlan,
	runs []db.TemplateRunDay,
	planID int32,
	start time.Time,
	markGoalRace bool,
) error {
	for i, tr := range runs {
		run, err := app.queries.CreateRunDay(ctx, db.CreateRunDayParams{
			PlanID:        planID,
			Date:          pgtype.Date{Time: start.AddDate(0, 0, int(tr.DayOffset)), Valid: true},
			RunType:       tr.RunType,
			TotalDistance: tr.Distance,
			DistanceUnit:  tmpl.DistanceUnit, // see note below
			TotalDuration: pgtype.Int8{Valid: false},
			Notes:         tr.Notes,
			IsGoalRace:    markGoalRace && tr.RunType == "race" && i == len(runs)-1,
		})
		if err != nil {
			return err
		}

		segs, err := app.queries.ListTemplateSegmentsByRun(ctx, tr.ID)
		if err != nil {
			return err
		}
		for _, ts := range segs {
			if _, err := app.queries.CreateSegment(ctx, db.CreateSegmentParams{
				RunID:          run.ID,
				OrderIndex:     ts.OrderIndex,
				Description:    ts.Description,
				EffortType:     ts.EffortType,
				Distance:       ts.Distance,
				DistanceUnit:   ts.DistanceUnit, // see note below
				Duration:       ts.Duration,
				Pace:           ts.Pace,
				Repetitions:    ts.Repetitions,
				HrZoneMin:      ts.HrZoneMin,
				HrZoneMax:      ts.HrZoneMax,
				HrAbsMin:       pgtype.Int4{},
				HrAbsMax:       pgtype.Int4{},
				SetIndex:       ts.SetIndex,
				SetRepetitions: ts.SetRepetitions,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func mondayOf(t time.Time) time.Time {
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	return t.AddDate(0, 0, -(wd - 1))
}
