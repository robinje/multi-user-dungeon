package main

import (
	"math/rand"

	"gonum.org/v1/gonum/stat/distuv"
)

// Challenge calculates the outcome of an action, adjusting the distribution based on the score difference to simulate the effect of kurtosis.
func Challenge(attackerScore, defenderScore float64) float64 {
	// Calculate the difference between the attacker's and defender's scores
	scoreDifference := attackerScore - defenderScore

	// Normalize the score difference to control the adjustment range
	// This ensures that the majority of interactions fall within a reasonable adjustment range,
	// with extreme cases being appropriately rare.
	normalizedDifference := scoreDifference / 10.0
	if normalizedDifference > 2 {
		normalizedDifference = 2
	} else if normalizedDifference < -2 {
		normalizedDifference = -2
	}

	// Adjust kurtosis based on the normalized score difference
	// Positive differences increase the right tail, negative differences increase the left tail.
	kurtosisAdjustment := normalizedDifference // Adjust this formula as needed for your game's mechanics

	// Create a modified normal distribution based on the kurtosis adjustment
	dist := distuv.Normal{
		Mu:    0,                        // Centered at 0
		Sigma: 1.0 + kurtosisAdjustment, // Adjust sigma to simulate kurtosis effect
	}

	// Generate a random value and calculate its corresponding z-score
	randomValue := rand.Float64()
	zScore := dist.Quantile(randomValue)

	// Ensure a minimum z-score of 0
	if zScore < 0 {
		zScore = 0
	}

	return zScore
}
