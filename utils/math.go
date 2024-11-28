package utils

import (
	"math"
)

func RoundTo(n float64, places int) float64 {
	return math.Round(n*math.Pow(10, float64(places))) / math.Pow(10, float64(places))
}

func ModF(n float64) float64 {
	out, _ := math.Modf(n)
	return out
}
