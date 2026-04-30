package hrutil

import "fmt"

type CalculatedZone struct {
	Number      int
	Name        string
	Description string
	Min         int
	Max         int
}

func FormatZoneRange(zone CalculatedZone) string {
	if zone.Max == 0 {
		return fmt.Sprintf("%d+ bpm", zone.Min)
	}
	return fmt.Sprintf("%d–%d bpm", zone.Min, zone.Max)
}

func ZoneMethodLabel(method string) string {
	switch method {
	case "lthr":
		return "Lactate Threshold HR (80/20 Fitzgerald)"
	case "hrr":
		return "Heart Rate Reserve (Karvonen)"
	default:
		return "Max Heart Rate"
	}
}
