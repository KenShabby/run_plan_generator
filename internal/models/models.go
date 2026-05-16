package models

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type PlanType string

const (
	Ultra        PlanType = "ultra"
	Marathon     PlanType = "marathon"
	HalfMarathon PlanType = "half_marathon"
	TenK         PlanType = "10k"
	FiveK        PlanType = "5k"
)

type RunType string

const (
	CruiseInterval      RunType = "cruise_interval"
	Easy                RunType = "easy"
	FastFinish          RunType = "fast_finish"
	Foundation          RunType = "foundation"
	GeneralAerobic      RunType = "general_aerobic"
	GeneralAerobicSpeed RunType = "general_aerobic_speed"
	HillRepetitions     RunType = "hill_repetitions"
	Interval            RunType = "interval"
	LactateThreshold    RunType = "lactate_threshold"
	LongInterval        RunType = "long_interval"
	LongRun             RunType = "long_run"
	LongRunFastFinish   RunType = "long_run_with_fast_finish"
	LongRunSpeedPlay    RunType = "long_run_with_speed_play"
	MediumLong          RunType = "medium_long"
	Race                RunType = "race"
	RacePace            RunType = "race_pace"
	Recovery            RunType = "recovery"
	RecoverySpeed       RunType = "recovery_speed"
	ShortInterval       RunType = "short_interval"
	SpeedPlay           RunType = "speed_play"
	Tempo               RunType = "tempo"
	VO2Max              RunType = "vo2_max"
)

type EffortType string // Are we running a given distance or given time?

const (
	ByDistance EffortType = "distance"
	ByTime     EffortType = "time"
)

type DistanceUnit string // Km or Freedom units?

const (
	Miles      DistanceUnit = "miles"
	Kilometers DistanceUnit = "kilometers"
)

// Handle HR Zone, range of zones, or absoulte HR range
type HeartRateTarget struct {
	ZoneMin int // 1-5
	ZoneMax int // 1-5, equals ZoneMin if single zone
	AbsMin  int // e.g. 132
	AbsMax  int // e.g. 145
}

type Segment struct {
	Order       int // The order this segment comes in the run
	Description string
	EffortType  EffortType
	Duration    time.Duration
	Distance    float64
	HeartRate   *HeartRateTarget // nil means no HR target specified
	Pace        time.Duration    // per unit defined by parent RunPlan
	Repetitions int
}

type Run struct {
	ID            int32
	PlanID        int32
	Date          time.Time
	Type          RunType
	TotalDistance float64
	TotalDuration time.Duration
	Completed     bool
	Notes         string
	Segments      []Segment
}

type RunPlan struct {
	ID           int32
	UserID       int32
	Name         string // e.g. Santa Rosa Marathon 2026
	Description  string
	PlanType     PlanType
	DistanceUnit DistanceUnit
	StartDate    time.Time
	EndDate      time.Time
	Runs         []Run
}

type User struct {
	ID        int32
	Email     string
	Username  string
	CreatedAt time.Time
}

type RPE int

const (
	RPECompleteRest RPE = 1
	RPEVeryEasy     RPE = 2
	RPEEasy         RPE = 3
	RPEModerate     RPE = 4
	RPESomewhatHard RPE = 5
	RPEHard         RPE = 6
	RPEVeryHard     RPE = 7
	RPEVeryVeryHard RPE = 8
	RPENearMaximal  RPE = 9
	RPEMaximal      RPE = 10
)

func (r RPE) Label() string {
	switch r {
	case RPECompleteRest:
		return "Complete Rest"
	case RPEVeryEasy:
		return "Very Easy"
	case RPEEasy:
		return "Easy"
	case RPEModerate:
		return "Moderate"
	case RPESomewhatHard:
		return "Somewhat Hard"
	case RPEHard:
		return "Hard"
	case RPEVeryHard:
		return "Very Hard"
	case RPEVeryVeryHard:
		return "Very Very Hard"
	case RPENearMaximal:
		return "Near Maximal"
	case RPEMaximal:
		return "Maximal"
	default:
		return "Unknown"
	}
}

// RPEOptions is used to populate the select in the UI
var RPEOptions = []struct {
	Value int
	Label string
}{
	{1, "1 - Complete Rest"},
	{2, "2 - Very Easy"},
	{3, "3 - Easy"},
	{4, "4 - Moderate"},
	{5, "5 - Somewhat Hard"},
	{6, "6 - Hard"},
	{7, "7 - Very Hard"},
	{8, "8 - Very Very Hard"},
	{9, "9 - Near Maximal"},
	{10, "10 - Maximal"},
}

// PaceFromDistanceAndDuration calculates pace in seconds per unit
// given distance and duration in seconds. Returns 0 if either is zero.
func PaceFromDistanceAndDuration(distance float64, durationSeconds int) int {
	if distance == 0 || durationSeconds == 0 {
		return 0
	}
	return int(float64(durationSeconds) / distance)
}

// FormatPace formats pace in seconds per mile as "M:SS /mi"
func FormatPace(paceSeconds int) string {
	if paceSeconds == 0 {
		return ""
	}
	mins := paceSeconds / 60
	secs := paceSeconds % 60
	return fmt.Sprintf("%d:%02d /mi", mins, secs)
}

// FormatDuration formats duration in seconds as "H:MM:SS" or "M:SS"
func FormatDuration(seconds int) string {
	if seconds == 0 {
		return ""
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// ParseDuration parses "H:MM:SS" or "MM:SS" into total seconds.
// Returns 0 and an error if the format is invalid.
func ParseDuration(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	parts := strings.Split(s, ":")
	switch len(parts) {
	case 2: // MM:SS
		m, err1 := strconv.Atoi(parts[0])
		sec, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			return 0, fmt.Errorf("invalid duration format, use MM:SS or H:MM:SS")
		}
		return m*60 + sec, nil
	case 3: // H:MM:SS
		h, err1 := strconv.Atoi(parts[0])
		m, err2 := strconv.Atoi(parts[1])
		sec, err3 := strconv.Atoi(parts[2])
		if err1 != nil || err2 != nil || err3 != nil {
			return 0, fmt.Errorf("invalid duration format, use MM:SS or H:MM:SS")
		}
		return h*3600 + m*60 + sec, nil
	default:
		return 0, fmt.Errorf("invalid duration format, use MM:SS or H:MM:SS")
	}
}

// ParsePace parses "M:SS" into seconds per mile.
// Returns 0 and an error if the format is invalid.
func ParsePace(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid pace format, use M:SS (e.g. 8:30)")
	}
	m, err1 := strconv.Atoi(parts[0])
	sec, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return 0, fmt.Errorf("invalid pace format, use M:SS (e.g. 8:30)")
	}
	return m*60 + sec, nil
}

// FormatPaceInput formats pace in seconds as "M:SS" suitable for form inputs.
func FormatPaceInput(paceSeconds int) string {
	if paceSeconds == 0 {
		return ""
	}
	mins := paceSeconds / 60
	secs := paceSeconds % 60
	return fmt.Sprintf("%d:%02d", mins, secs)
}
