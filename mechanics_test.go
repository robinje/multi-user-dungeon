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
		expectedRange [2]float64 // Lower and upper bounds for expected result
	}{
		{"EqualScores", 5.0, 5.0, [2]float64{-0.5, 0.5}},        // Expecting near-zero adjustment for equal scores
		{"AttackerAdvantage", 7.0, 5.0, [2]float64{0.1, 0.3}},   // Expecting positive adjustment for attacker advantage
		{"DefenderAdvantage", 5.0, 7.0, [2]float64{-0.3, -0.1}}, // Expecting negative adjustment for defender advantage
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call the Challenge function with test case parameters
			outcome := Challenge(tc.attackerScore, tc.defenderScore)

			// Check if the outcome is within the expected range
			if outcome < tc.expectedRange[0] || outcome > tc.expectedRange[1] {
				t.Errorf("Outcome %.4f for %v outside of expected range %.4f to %.4f", outcome, tc.name, tc.expectedRange[0], tc.expectedRange[1])
			}
		})
	}
}
