package pages

import (
	"time"

	"github.com/KenShabby/run_plan_generator/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

// WeekRow holds 7 day slots (Mon-Sun). A nil entry means no run that day.
type WeekRow struct {
	WeekNum int
	Days    [7]*db.RunDay // index 0=Mon, 6=Sun
}

func GroupRunsByWeek(runs []db.RunDay, startDate time.Time) []WeekRow {
	if len(runs) == 0 {
		return nil
	}

	// Normalise startDate to Monday of its week
	weekday := int(startDate.Weekday()) // Sunday=0
	if weekday == 0 {
		weekday = 7
	}
	monday := startDate.AddDate(0, 0, -(weekday - 1))

	// Find last run date
	last := runs[len(runs)-1].Date.Time
	totalWeeks := int(last.Sub(monday).Hours()/24)/7 + 1

	rows := make([]WeekRow, totalWeeks)
	for w := range rows {
		rows[w].WeekNum = w + 1
	}

	for i := range runs {
		r := &runs[i]
		diff := int(r.Date.Time.Sub(monday).Hours() / 24)
		week := diff / 7
		day := diff % 7
		if week >= 0 && week < totalWeeks && day >= 0 && day < 7 {
			rows[week].Days[day] = r
		}
	}
	return rows
}

func mustFloat(n pgtype.Numeric) float64 {
	f, _ := n.Float64Value()
	return f.Float64
}
