package main

import (
	"math"

	"github.com/KenShabby/run_plan_generator/internal/hrutil"
)

type ZoneDefinition struct {
	Number      int
	Name        string
	Description string
	MinPct      float64
	MaxPct      float64 // 0 means no upper bound
}

// Fitzgerald 80/20 LTHR-based zones
var lthrZones = []ZoneDefinition{
	{1, "Zone 1", "Recovery", 0.75, 0.80},
	{2, "Zone 2", "Aerobic", 0.81, 0.89},
	{3, "Zone 3", "Tempo", 0.96, 1.00},
	{4, "Zone 4", "Threshold", 1.02, 1.05},
	{5, "Zone 5", "VO2 Max", 1.06, 0},
}

// Standard HRR (Karvonen) zones
var hrrZones = []ZoneDefinition{
	{1, "Zone 1", "Recovery", 0.50, 0.60},
	{2, "Zone 2", "Aerobic", 0.60, 0.70},
	{3, "Zone 3", "Tempo", 0.70, 0.80},
	{4, "Zone 4", "Threshold", 0.80, 0.90},
	{5, "Zone 5", "VO2 Max", 0.90, 1.00},
}

// Standard Max HR zones
var maxHRZones = []ZoneDefinition{
	{1, "Zone 1", "Recovery", 0.50, 0.60},
	{2, "Zone 2", "Aerobic", 0.60, 0.70},
	{3, "Zone 3", "Tempo", 0.70, 0.80},
	{4, "Zone 4", "Threshold", 0.80, 0.90},
	{5, "Zone 5", "VO2 Max", 0.90, 1.00},
}

func calculateZones(maxHR, restingHR, lthr int, method string) []hrutil.CalculatedZone {
	switch method {
	case "lthr":
		return calcLTHRZones(lthr, maxHR)
	case "hrr":
		return calcHRRZones(maxHR, restingHR)
	default:
		return calcMaxHRZones(maxHR)
	}
}

func calcLTHRZones(lthr, maxHR int) []hrutil.CalculatedZone {
	zones := make([]hrutil.CalculatedZone, len(lthrZones))
	for i, z := range lthrZones {
		min := int(math.Round(float64(lthr) * z.MinPct))
		max := 0
		if z.MaxPct > 0 {
			max = int(math.Round(float64(lthr) * z.MaxPct))
		} else {
			max = maxHR
		}
		zones[i] = hrutil.CalculatedZone{
			Number:      z.Number,
			Name:        z.Name,
			Description: z.Description,
			Min:         min,
			Max:         max,
		}
	}
	return zones
}

func calcHRRZones(maxHR, restingHR int) []hrutil.CalculatedZone {
	hrr := maxHR - restingHR
	zones := make([]hrutil.CalculatedZone, len(hrrZones))
	for i, z := range hrrZones {
		min := int(math.Round(float64(hrr)*z.MinPct)) + restingHR
		max := int(math.Round(float64(hrr)*z.MaxPct)) + restingHR
		zones[i] = hrutil.CalculatedZone{
			Number:      z.Number,
			Name:        z.Name,
			Description: z.Description,
			Min:         min,
			Max:         max,
		}
	}
	return zones
}

func calcMaxHRZones(maxHR int) []hrutil.CalculatedZone {
	zones := make([]hrutil.CalculatedZone, len(maxHRZones))
	for i, z := range maxHRZones {
		min := int(math.Round(float64(maxHR) * z.MinPct))
		max := int(math.Round(float64(maxHR) * z.MaxPct))
		zones[i] = hrutil.CalculatedZone{
			Number:      z.Number,
			Name:        z.Name,
			Description: z.Description,
			Min:         min,
			Max:         max,
		}
	}
	return zones
}

func bestMethod(maxHR, restingHR, lthr int) string {
	if lthr > 0 {
		return "lthr"
	}
	if restingHR > 0 {
		return "hrr"
	}
	return "max_hr"
}
