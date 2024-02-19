package main

import (
	"math"
	"math/rand"
)

func Challenge(attacker, defender float64) float64 {
	// Calculate the difference to determine the shift
	diff := attacker - defender

	// Assuming a default steepness k=1 for simplicity; adjust k if a different steepness is desired
	k := 0.25

	// Simplified sigmoid function evaluation at x=0 with shift
	sigmoidValue := 1 / (1 + math.Exp(k*diff))

	// Generate a random float64 number
	randomNumber := rand.Float64()

	// Divide the random number by the sigmoid value
	result := randomNumber / sigmoidValue

	return result
}
