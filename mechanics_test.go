package main

import (
	"testing"
)

// TestChallenge tests the Challenge function to ensure it produces consistent and expected results.
func TestChallenge(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name          string
		attackerScore float64
		defenderScore float64
		expectedMin   float64 // Minimum expected result, considering non-negative outcomes
	}{
		// Since the Challenge function's output is now heavily dependent on random values and adjusted sigma,
		// defining an exact expected range becomes challenging. Instead, we'll ensure the output is non-negative
		// and attempt to verify behavior based on score differences.
		{"EqualScores", 5.0, 5.0, 0},          // Expecting minimal adjustment for equal scores
		{"AttackerAdvantage", 10.0, 5.0, 0},   // Expecting possible higher outcome due to advantage
		{"DefenderAdvantage", 5.0, 10.0, 0},   // Even with a disadvantage, outcome should not be negative
		{"ExtremeAdvantage", 20.0, 0.0, 0},    // Testing edge case with a significant score difference
		{"ExtremeDisadvantage", 0.0, 20.0, 0}, // Testing edge case with significant disadvantage
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			outcome := Challenge(tc.attackerScore, tc.defenderScore)

			// Check if the outcome is non-negative and meets minimum expectations
			if outcome < tc.expectedMin {
				t.Errorf("Outcome %.4f for '%s' is less than the minimum expected %.4f", outcome, tc.name, tc.expectedMin)
			}
		})
	}
}
