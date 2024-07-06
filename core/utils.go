package core

import (
	"math"
	"math/rand"
)

func Challenge(attacker, defender, balance float64) float64 {
	// Calculate the difference to determine the shift
	diff := attacker - defender

	// Simplified sigmoid function evaluation at x=0 with shift
	sigmoidValue := 1 / (1 + math.Exp(balance*diff))

	// Generate a random float64 number
	randomNumber := rand.Float64()

	// Divide the random number by the sigmoid value
	result := randomNumber / sigmoidValue

	return result
}
