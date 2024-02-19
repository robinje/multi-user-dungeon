package main

import (
	"testing"
)

// TestPositiveDifference checks the function with attacker > defender
func TestPositiveDifference(t *testing.T) {
	attacker, defender := 10.0, 5.0
	result := Challenge(attacker, defender)
	if result <= 0 {
		t.Errorf("Expected a positive result, got %v", result)
	}
}

// TestNegativeDifference checks the function with attacker < defender
func TestNegativeDifference(t *testing.T) {
	attacker, defender := 5.0, 10.0
	result := Challenge(attacker, defender)
	if result <= 0 {
		t.Errorf("Expected a positive result, got %v", result)
	}
}

// TestZeroDifference checks the function with attacker == defender
func TestZeroDifference(t *testing.T) {
	attacker, defender := 5.0, 5.0
	result := Challenge(attacker, defender)
	if result <= 0 {
		t.Errorf("Expected a positive result, got %v", result)
	}
}

// TestWithExtremes checks the function with inputs at the extremes of the valid range
func TestWithExtremes(t *testing.T) {
	tests := []struct {
		attacker float64
		defender float64
	}{
		{0, 20},
		{20, 0},
	}

	for _, test := range tests {
		result := Challenge(test.attacker, test.defender)
		if result <= 0 {
			t.Errorf("Expected a positive result for attacker = %v and defender = %v, got %v", test.attacker, test.defender, result)
		}
	}
}

// TestBoundaryValues checks the function with boundary values
func TestBoundaryValues(t *testing.T) {
	tests := []struct {
		attacker float64
		defender float64
	}{
		{0, 0},
		{20, 20},
	}

	for _, test := range tests {
		result := Challenge(test.attacker, test.defender)
		if result <= 0 {
			t.Errorf("Expected a positive result for attacker = %v and defender = %v, got %v", test.attacker, test.defender, result)
		}
	}
}

// TestStatisticalDistribution tests the distribution of outcomes over many runs
func TestStatisticalDistribution(t *testing.T) {
	attacker, defender := 15.0, 5.0
	const runs = 1000
	results := make([]float64, runs)

	for i := 0; i < runs; i++ {
		results[i] = Challenge(attacker, defender)
	}

	var sum float64
	for _, result := range results {
		sum += result
		if result <= 0 {
			t.Errorf("Expected a positive result, got %v", result)
		}
	}
	average := sum / float64(runs)
	if average <= 0 {
		t.Errorf("Expected a positive average result, got %v", average)
	}
}

// TestStabilityWithConstantInputs checks if repeated calls with the same inputs yield different outcomes
func TestStabilityWithConstantInputs(t *testing.T) {
	attacker, defender := 10.0, 10.0
	firstResult := Challenge(attacker, defender)
	secondResult := Challenge(attacker, defender)

	if firstResult == secondResult {
		t.Errorf("Expected different results for repeated calls, got %v and %v", firstResult, secondResult)
	}
}
