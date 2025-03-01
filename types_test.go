package main

import (
	"testing"

	"github.com/bcicen/go-units"
)

func TestMilesPerHourConversion(t *testing.T) {
	mps := units.NewValue(1, MilesPerHour)
	converted, err := mps.Convert(MetersPerSecond)
	if err != nil {
		t.Fatalf("failed to convert 1mps to ms: %s", err)
	}

	result := converted.Float()
	if result != 0.44704 {
		t.Fatalf("expected 0.44704, got %f\n", result)
	}
}
