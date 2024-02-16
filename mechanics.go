package main

import (
	"math/rand"
	"time"

	"gonum.org/v1/gonum/stat/distuv"
)

// calculates the outcome of an action, adjusting for kurtosis based on the score difference.
func Challenge(attackerScore, defenderScore float64) float64 {

	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	// Calculate the difference between the attacker's and defender's scores
	scoreDifference := attackerScore - defenderScore

	// Determine the kurtosis adjustment based on the score difference
	// This conceptual adjustment factor simulates the effect of kurtosis based on the score difference.
	kurtosisAdjustment := scoreDifference / 10.0

	// Create a standard normal distribution
	dist := distuv.UnitNormal

	// Generate a random percentile (0 to 1) to find a corresponding z-score
	randomPercentile := rand.Float64()

	// Calculate the z-score for the random percentile
	zScore := dist.Quantile(randomPercentile)

	// Adjust the z-score based on the kurtosis adjustment
	// The adjustment simulates the effect of kurtosis by scaling the outcome according to the score difference.
	adjustedZScore := zScore + kurtosisAdjustment

	return adjustedZScore
}
