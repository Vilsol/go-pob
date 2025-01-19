package utils

import (
	"math"
)

func RoundTo(n float64, places int) float64 {
	multiplier := math.Pow(10, float64(places))
	return math.Round(n*multiplier) / multiplier
}

func ModF(n float64) float64 {
	out, _ := math.Modf(n)
	return out
}

func FloorTo(n float64, places int) float64 {
	multiplier := math.Pow10(places)
	return math.Floor(n*multiplier) / multiplier
}
