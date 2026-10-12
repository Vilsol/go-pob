package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

//crystalline:exports
func PartyExports(r bind.Registry) {
	r.Type(calculator.PartySummaryEntry{}, bind.Plain())
	r.Type(calculator.GemSupportPair{}, bind.Plain())

	r.Func(GetPartySummaries)
	r.Func(MiscGemSupportPairs)
}

func GetPartySummaries(build *pob.PathOfBuilding) map[string][]calculator.PartySummaryEntry {
	return calculator.PartySummaries(build.Party)
}

func MiscGemSupportPairs(m *MiscCalculator, group int) []calculator.GemSupportPair {
	if m == nil || m.calc == nil {
		return nil
	}
	return m.calc.Base().GemSupportPairs(group)
}
