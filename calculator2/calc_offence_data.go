package calculator2

import (
	"sync"

	game "github.com/Vilsol/go-pob-data/poe2"
)

// costDivisors is data.costs[resource].Divisor, from the game's CostTypes table.
//
//pob2:block:src/Modules/Data.lua:721-729
var costDivisors = sync.OnceValues(func() (map[string]float64, error) {
	rows, err := loadRows[game.CostType](gameDataContext(), game.CostTypesTable, gameDataCache())
	if err != nil {
		return nil, err
	}
	out := make(map[string]float64, len(rows))
	for _, row := range rows {
		out[row.ID] = float64(row.Divisor)
	}
	return out, nil
})

// costDivisor is data.costs[resource].Divisor; a resource the table lacks makes PoB index nil, which
// the cost loop only reaches for a skill cost, so it is a data defect and divides by 1.
func costDivisor(resource string) float64 {
	divisors, err := costDivisors()
	if err != nil {
		return 1
	}
	if d, ok := divisors[resource]; ok && d != 0 {
		return d
	}
	return 1
}
