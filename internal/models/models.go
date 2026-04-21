package models

import "time"

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
	CruiseInterval    RunType = "cruise_interval"
	Easy              RunType = "easy"
	FastFinish        RunType = "fast_finish"
	Foundation        RunType = "foundation"
	GeneralAerobic    RunType = "general_aerobic"
	HillRepetitions   RunType = "hill_repetitions"
	Interval          RunType = "interval"
	LactateThreshold  RunType = "lactate_threshold"
	LongInterval      RunType = "long_interval"
	LongRun           RunType = "long_run"
	LongRunFastFinish RunType = "long_run_with_fast_finish"
	LongRunSpeedPlay  RunType = "long_run_with_speed_play"
	MediumLong        RunType = "medium_long"
	Race              RunType = "race"
	RacePace          RunType = "race_pace"
	Recovery          RunType = "recovery"
	ShortInterval     RunType = "short_interval"
	SpeedPlay         RunType = "speed_play"
	Tempo             RunType = "tempo"
	VO2Max            RunType = "vo2_max"
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
	Neme         string // e.g. Santa Rosa Marathon 2026
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
