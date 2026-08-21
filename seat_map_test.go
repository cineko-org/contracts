package contracts

import (
	"errors"
	"testing"
)

func TestSeatMapResolutionValidation(t *testing.T) {
	valid := []SeatMapResolution{
		{Status: SeatMapResolutionReady, SeatMap: &SeatMapVersion{}},
		{Status: SeatMapResolutionWaiting},
		{Status: SeatMapResolutionUnverifiable},
	}
	for _, resolution := range valid {
		if err := resolution.Validate(); err != nil {
			t.Fatalf("Validate(%q) error = %v", resolution.Status, err)
		}
	}

	invalid := []SeatMapResolution{
		{Status: SeatMapResolutionReady},
		{Status: SeatMapResolutionWaiting, SeatMap: &SeatMapVersion{}},
		{Status: SeatMapResolutionUnverifiable, SeatMap: &SeatMapVersion{}},
		{Status: "unknown"},
	}
	for _, resolution := range invalid {
		if err := resolution.Validate(); !errors.Is(err, ErrUnsupportedSeatMapResolution) {
			t.Fatalf("Validate(%q) error = %v", resolution.Status, err)
		}
	}
}
