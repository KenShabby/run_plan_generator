package main

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/KenShabby/run_plan_generator/internal/db"
)

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

func formatSegmentsText(segments []db.Segment) string {
	if len(segments) == 0 {
		return ""
	}

	var sb strings.Builder
	groups := groupSegmentsForICS(segments)

	for _, group := range groups {
		if group.isSet {
			sb.WriteString(fmt.Sprintf("Repeat %d×:\\n", group.repetitions))
			for _, seg := range group.segments {
				sb.WriteString("  ")
				sb.WriteString(formatSegmentLine(seg))
				sb.WriteString("\\n")
			}
		} else {
			sb.WriteString(formatSegmentLine(group.segments[0]))
			sb.WriteString("\\n")
		}
	}

	return sb.String()
}

func formatSegmentLine(seg db.Segment) string {
	var parts []string

	if seg.Description.Valid && seg.Description.String != "" {
		parts = append(parts, seg.Description.String)
	}
	if seg.Distance.Valid {
		parts = append(parts, fmt.Sprintf("%.2f mi", seg.Distance.Float64))
	}
	if seg.Duration.Valid {
		secs := seg.Duration.Int64
		parts = append(parts, fmt.Sprintf("%d:%02d", secs/60, secs%60))
	}
	if seg.HrZoneMin.Valid && seg.HrZoneMax.Valid {
		if seg.HrZoneMin.Int32 == seg.HrZoneMax.Int32 {
			parts = append(parts, fmt.Sprintf("Z%d", seg.HrZoneMin.Int32))
		} else {
			parts = append(parts, fmt.Sprintf("Z%d–Z%d", seg.HrZoneMin.Int32, seg.HrZoneMax.Int32))
		}
	}

	return strings.Join(parts, " · ")
}

// local version of the grouping logic for plain structs, avoids importing pages package
type icsSegmentGroup struct {
	isSet       bool
	repetitions int32
	segments    []db.Segment
}

func groupSegmentsForICS(segments []db.Segment) []icsSegmentGroup {
	var groups []icsSegmentGroup
	seenSets := map[int32]bool{}

	for _, seg := range segments {
		if !seg.SetIndex.Valid {
			groups = append(groups, icsSegmentGroup{
				isSet:    false,
				segments: []db.Segment{seg},
			})
			continue
		}

		setIdx := seg.SetIndex.Int32
		if seenSets[setIdx] {
			for i := range groups {
				if groups[i].isSet && groups[i].segments[0].SetIndex.Int32 == setIdx {
					groups[i].segments = append(groups[i].segments, seg)
					break
				}
			}
			continue
		}

		seenSets[setIdx] = true
		groups = append(groups, icsSegmentGroup{
			isSet:       true,
			repetitions: seg.SetRepetitions.Int32,
			segments:    []db.Segment{seg},
		})
	}

	return groups
}

func buildICS(plan db.TrainingPlan, runs []db.RunDay, segmentsByRun map[int32][]db.Segment) string {
	var sb strings.Builder

	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString("PRODID:-//Run Plan Generator//EN\r\n")
	sb.WriteString("CALSCALE:GREGORIAN\r\n")
	sb.WriteString("METHOD:PUBLISH\r\n")

	for _, run := range runs {
		if !run.Date.Valid {
			continue
		}

		date := run.Date.Time.Format("20060102")
		uid := fmt.Sprintf("run-%d@run_plan_generator", run.ID)
		summary := formatRunType(run.RunType)
		now := time.Now().UTC().Format("20060102T150405Z")

		var descParts []string
		if run.TotalDistance.Valid {
			descParts = append(descParts, fmt.Sprintf("%.1f mi", run.TotalDistance.Float64))
		}
		if run.Notes.Valid && run.Notes.String != "" {
			descParts = append(descParts, run.Notes.String)
		}
		if segs, ok := segmentsByRun[run.ID]; ok && len(segs) > 0 {
			descParts = append(descParts, "\\n"+formatSegmentsText(segs))
		}

		description := strings.Join(descParts, "\\n")

		sb.WriteString("BEGIN:VEVENT\r\n")
		sb.WriteString(fmt.Sprintf("UID:%s\r\n", uid))
		sb.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", now))
		sb.WriteString(fmt.Sprintf("DTSTART;VALUE=DATE:%s\r\n", date))
		sb.WriteString(fmt.Sprintf("DTEND;VALUE=DATE:%s\r\n", date))
		sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", summary))
		if description != "" {
			sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", description))
		}
		sb.WriteString("END:VEVENT\r\n")
	}

	sb.WriteString("END:VCALENDAR\r\n")
	return sb.String()
}
