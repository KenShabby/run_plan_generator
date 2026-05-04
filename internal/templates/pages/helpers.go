package pages

import (
	"strings"
	"unicode"

	"github.com/KenShabby/run_plan_generator/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

// WeekRow holds 7 day slots (Mon-Sun). A nil entry means no run that day.
type WeekRow struct {
	WeekNum int
	Days    [7]*db.RunDay // index 0=Mon, 6=Sun
}

type SegmentGroup struct {
	IsSet       bool
	Repetitions int32
	Segments    []db.Segment
}

func GroupSegments(segments []db.Segment) []SegmentGroup {
	var groups []SegmentGroup
	// track which set_indexes we've already processed
	seenSets := map[int32]bool{}

	for _, seg := range segments {
		if !seg.SetIndex.Valid {
			// standalone segment
			groups = append(groups, SegmentGroup{
				IsSet:    false,
				Segments: []db.Segment{seg},
			})
			continue
		}

		setIdx := seg.SetIndex.Int32
		if seenSets[setIdx] {
			// already added this set's group, find it and append
			for i := range groups {
				if groups[i].IsSet && groups[i].Segments[0].SetIndex.Int32 == setIdx {
					groups[i].Segments = append(groups[i].Segments, seg)
					break
				}
			}
			continue
		}

		// first time seeing this set
		seenSets[setIdx] = true
		groups = append(groups, SegmentGroup{
			IsSet:       true,
			Repetitions: seg.SetRepetitions.Int32,
			Segments:    []db.Segment{seg},
		})
	}

	return groups
}

func GroupRunsByWeek(runs []db.RunDay) []WeekRow {
	if len(runs) == 0 {
		return nil
	}

	// Anchor to Monday of the week containing the first run
	firstRun := runs[0].Date.Time
	weekday := int(firstRun.Weekday()) // Sunday=0
	if weekday == 0 {
		weekday = 7
	}
	monday := firstRun.AddDate(0, 0, -(weekday - 1))

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

func formatRunType(runType string) string {
	words := strings.Split(runType, "_")
	for i, w := range words {
		if len(w) == 0 {
			continue
		}
		runes := []rune(w)
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}
