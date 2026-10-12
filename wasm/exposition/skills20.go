package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

//crystalline:exports
func Skills20Exports(r bind.Registry) {
	r.Func(GemDefaultLevelMode)
	r.Func(GemDefaultLevel)
	r.Func(SocketedInItemNames, bind.AsPromise())
	r.Type(calculator.SpectreRow{}, bind.Plain())
	r.Func(SpectreLibrary)
	r.Func(SpectreRows)
	r.Func(CalcGemStats, bind.AsPromise())
	r.Func(CalcConfigStats, bind.AsPromise())
}

func GemDefaultLevelMode(build *pob.PathOfBuilding) string {
	return calculator.DefaultGemLevelMode(build.Skills)
}

func GemDefaultLevel(build *pob.PathOfBuilding, gemID string) int {
	return calculator.ProcessGemLevel(build, gemID, false)
}

func SocketedInItemNames(build *pob.PathOfBuilding, slotNames []string) map[string]string {
	out := make(map[string]string, len(slotNames))
	for _, slot := range slotNames {
		if name := calculator.ActiveSetSlotItemName(build, slot); name != "" {
			out[slot] = name
		}
	}
	return out
}

func SpectreLibrary() []calculator.SpectreRow {
	return calculator.SpectreLibrary()
}

func SpectreRows(ids []string) []calculator.SpectreRow {
	return calculator.SpectreRows(ids)
}

func CalcGemStats(m *MiscCalculator, socketGroup int, gemIndex int, gem pob.Gem) StatOutputs {
	return statOutputs(m.calc.CalcGem(socketGroup, gemIndex, gem))
}

func CalcConfigStats(m *MiscCalculator, input pob.Input, remove bool) StatOutputs {
	return statOutputs(m.calc.CalcConfig(input, remove))
}
