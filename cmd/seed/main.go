package main

/* This is a helper app to convert yaml run templates into db run_plans */

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/KenShabby/run_plan_generator/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"gopkg.in/yaml.v3"
)

type SegmentYAML struct {
	OrderIndex     int     `yaml:"order_index"` // Order in the run
	Description    string  `yaml:"description"`
	EffortType     string  `yaml:"effort_type"`
	Distance       float64 `yaml:"distance"`
	Duration       int     `yaml:"duration"`
	Repetitions    int     `yaml:"repetitions"`
	HrZoneMin      int     `yaml:"hr_zone_min"`
	HrZoneMax      int     `yaml:"hr_zone_max"`
	SetIndex       int     `yaml:"set_index"` // Order in the repetitions
	SetRepetitions int     `yaml:"set_repetitions"`
}

type RunDayYAML struct {
	DayOffset int           `yaml:"day_offset"`
	RunType   string        `yaml:"run_type"`
	Distance  float64       `yaml:"distance"`
	Notes     string        `yaml:"notes"`
	Segments  []SegmentYAML `yaml:"segments"`
}

type TemplatePlanYAML struct {
	Name              string       `yaml:"name"`
	Description       string       `yaml:"description"`
	PlanType          string       `yaml:"plan_type"`
	DistanceUnit      string       `yaml:"distance_unit"`
	TotalWeeks        int          `yaml:"total_weeks"`
	PeakWeeklyMileage float64      `yaml:"peak_weekly_mileage"`
	Runs              []RunDayYAML `yaml:"runs"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: seed <yaml_file>")
	}

	connStr := getEnv("DATABASE_URL", "postgres:///run_plan_generator?sslmode=disable")
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatalf("unable to connect: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("unable to reach database: %v", err)
	}

	queries := db.New(pool)

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalf("unable to read file: %v", err)
	}

	var plan TemplatePlanYAML
	if err := yaml.Unmarshal(data, &plan); err != nil {
		log.Fatalf("unable to parse yaml: %v", err)
	}

	ctx := context.Background()

	// Insert the template plan
	tmpl, err := queries.CreateTemplatePlan(ctx, db.CreateTemplatePlanParams{
		Name:              plan.Name,
		Description:       pgtype.Text{String: plan.Description, Valid: plan.Description != ""},
		PlanType:          plan.PlanType,
		DistanceUnit:      plan.DistanceUnit,
		TotalWeeks:        int32(plan.TotalWeeks),
		PeakWeeklyMileage: pgtype.Numeric{Valid: false},
	})
	if err != nil {
		log.Fatalf("failed to create template plan: %v", err)
	}
	fmt.Printf("created template plan: %s (id=%d)\n", tmpl.Name, tmpl.ID)

	// Insert each run day and its segments
	for _, run := range plan.Runs {
		var dist pgtype.Float8
		if run.Distance > 0 {
			dist = pgtype.Float8{Float64: run.Distance, Valid: true}
		}

		tmplRun, err := queries.CreateTemplateRunDay(ctx, db.CreateTemplateRunDayParams{
			PlanID:    tmpl.ID,
			DayOffset: int32(run.DayOffset),
			RunType:   run.RunType,
			Distance:  dist,
			Notes:     pgtype.Text{String: run.Notes, Valid: run.Notes != ""},
		})
		if err != nil {
			log.Fatalf("failed to create run day (offset %d): %v", run.DayOffset, err)
		}
		fmt.Printf("  created run: day_offset=%d type=%s\n", tmplRun.DayOffset, tmplRun.RunType)

		// Insert segments for this run
		for _, seg := range run.Segments {
			var dist pgtype.Float8
			var dur pgtype.Int8

			if seg.Duration > 0 {
				dur = pgtype.Int8{Int64: int64(seg.Duration), Valid: true}
			}
			if seg.Distance > 0 {
				dist = pgtype.Float8{Float64: seg.Distance, Valid: true}
			}
			reps := seg.Repetitions
			if reps == 0 {
				reps = 1
			}
			var hrMin, hrMax pgtype.Int4
			if seg.HrZoneMin > 0 {
				hrMin = pgtype.Int4{Int32: int32(seg.HrZoneMin), Valid: true}
			}
			if seg.HrZoneMax > 0 {
				hrMax = pgtype.Int4{Int32: int32(seg.HrZoneMax), Valid: true}
			}

			var setIndex, setReps pgtype.Int4
			if seg.SetIndex > 0 {
				setIndex = pgtype.Int4{Int32: int32(seg.SetIndex), Valid: true}
				setReps = pgtype.Int4{Int32: int32(seg.SetRepetitions), Valid: true}
			}

			_, err := queries.CreateTemplateSegment(ctx, db.CreateTemplateSegmentParams{
				RunID:      tmplRun.ID,
				OrderIndex: int32(seg.OrderIndex),
				Description: pgtype.Text{
					String: seg.Description,
					Valid:  seg.Description != "",
				},
				EffortType:     seg.EffortType,
				Distance:       dist,
				Duration:       dur,
				Pace:           pgtype.Int8{Valid: false},
				Repetitions:    int32(reps),
				HrZoneMin:      hrMin,
				HrZoneMax:      hrMax,
				SetIndex:       setIndex,
				SetRepetitions: setReps,
			})
			if err != nil {
				log.Fatalf("failed to create segment: %v", err)
			}
			fmt.Printf("    created segment: %s\n", seg.Description)
		}
	}

	fmt.Printf("\ndone — seeded %d runs\n", len(plan.Runs))
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
