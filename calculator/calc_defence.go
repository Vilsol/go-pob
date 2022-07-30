package calculator

import "math"

func CalcArmourReductionF(armour float64, raw float64) float64 {
	if armour == 0 && raw == 0 {
		return 0
	}
	return armour / (armour + raw*5) * 100
}

func CalcArmourReduction(armour float64, raw float64) float64 {
	return math.Round(CalcArmourReductionF(armour, raw))
}

func CalcHitChance(evasion float64, accuracy float64) float64 {
	if accuracy < 0 {
		return 5
	}
	rawChance := accuracy / (accuracy + math.Pow(evasion/5, 0.9)) * 125
	return math.Max(math.Min(math.Round(rawChance), 100), 5)
}
