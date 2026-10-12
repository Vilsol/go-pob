package exposition

import (
	"maps"
	"slices"
	"strconv"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

type StatOutputs struct {
	Output          map[string]float64
	Minion          map[string]float64
	SkillFlags      map[calculator.SkillFlag]bool
	GrantedPassives []int64
}

type MiscCalculator struct {
	calc *calculator.MiscCalculator
	base StatOutputs
}

func GetMiscCalculator(build *pob.PathOfBuilding) *MiscCalculator {
	calc := calculator.GetMiscCalculator(build)
	return &MiscCalculator{calc: calc, base: statOutputs(calc.Base())}
}

func MiscCalculatorFor(build *pob.PathOfBuilding, calc *calculator.Calculator) *MiscCalculator {
	misc := calculator.MiscCalculatorOver(build, calc)
	if misc == nil {
		return nil
	}
	out := statOutputs(misc.Base())
	out.Output = maps.Clone(out.Output)
	for _, k := range calculator.MainOnlyOutputs {
		delete(out.Output, k)
	}
	return &MiscCalculator{calc: misc, base: out}
}

func (m *MiscCalculator) Base() StatOutputs {
	return m.base
}

func (m *MiscCalculator) CalcNodes(addNodes []int64, removeNodes []int64) StatOutputs {
	return statOutputs(m.calc.Calc(&calculator.CalcOverride{AddNodes: addNodes, RemoveNodes: removeNodes}))
}

func statOutputs(env *calculator.Environment) StatOutputs {
	out := StatOutputs{Output: env.Player.Output}
	if env.Player.MainSkill != nil {
		out.SkillFlags = env.Player.MainSkill.SkillFlags
	}
	if env.Minion != nil {
		out.Minion = env.Minion.Output
	}
	for id := range env.GrantedPassives {
		if n, err := strconv.ParseInt(id, 10, 64); err == nil {
			out.GrantedPassives = append(out.GrantedPassives, n)
		}
	}
	slices.Sort(out.GrantedPassives)
	return out
}
