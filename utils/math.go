package utils

import (
	"math"

	"golang.org/x/exp/constraints"
)

func Min[T constraints.Ordered](values ...T) T {
	min := values[0]

	for _, value := range values[1:] {
		if value < min {
			min = value
		}
	}

	return min
}

func Max[T constraints.Ordered](values ...T) T {
	max := values[0]

	for _, value := range values[1:] {
		if value > max {
			max = value
		}
	}

	return max
}

func RoundTo(n float64, places int) float64 {
	return math.Round(n*math.Pow(10, float64(places))) / math.Pow(10, float64(places))
}
