package calculator

import "runtime"

var sortValuesRollUpAll bool

//pob:block:src/Modules/Calcs.lua:134-147
func (m *MiscCalculator) sortValues(slotName string, texts []string, ok []bool, ps PowerStat) []float64 {
	out := make([]float64, len(texts))
	seen := make(map[string]float64, len(texts))
	for k, text := range texts {
		if !ok[k] {
			continue
		}
		if v, hit := seen[text]; hit {
			out[k] = v
			continue
		}
		override := &CalcOverride{RepSlotName: slotName, RepItem: ParseItem(text)}
		var env *Environment
		if ps.Stat == "FullDPS" || sortValuesRollUpAll {
			env = m.Calc(override)
		} else {
			runtime.Gosched()
			env, _, _, _ = InitEnvOverride(m.build, m.cache, OutputModeCalculator, override)
			PerformCalc(env)
		}
		out[k] = powerStatValue(env, ps)
		seen[text] = out[k]
	}
	return out
}
