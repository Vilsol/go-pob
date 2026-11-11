package exposition

import (
	"maps"

	"github.com/Vilsol/go-pob/calculator2"
)

// ConfigPlaceholders is each option control's placeholder after the last calculation.
func (c *Calculator) ConfigPlaceholders() map[string]float64 {
	if c.env == nil {
		return nil
	}
	return calculator2.ConfigPlaceholders(c.env)
}

// ConfigCompare is an option tooltip's stat differences: the option's input set to value.
func (c *Calculator) ConfigCompare(name string, value calculator2.ConfigValue) (*ItemCompareView, error) {
	return c.miscCompare(func(m *calculator2.MiscCalculator) *calculator2.Environment { return m.CalcConfig(name, value) })
}

// CustomModGroupCompare is the stat differences of toggling custom modifier group i.
func (c *Calculator) CustomModGroupCompare(i int) (*ItemCompareView, error) {
	return c.miscCompare(func(m *calculator2.MiscCalculator) *calculator2.Environment { return m.CalcCustomModGroup(i) })
}

// miscCompare is one calculation's stat differences over the last calculation, as one headerless block.
func (c *Calculator) miscCompare(calc func(*calculator2.MiscCalculator) *calculator2.Environment) (*ItemCompareView, error) {
	misc, err := c.miscCalculator()
	if err != nil {
		return nil, err
	}
	view := &ItemCompareView{Base: flaggedOutputs(misc.Base), SkillFlags: statSetFlags(c.env.Player.MainSkill), Blocks: []ItemCompareBlock{}}
	if main := c.env.Player.MainSkill; main != nil && main.Minion != nil {
		view.Minion = true
		view.MinionSkillFlags = statSetFlags(main.Minion.MainSkill)
	}
	if env := calc(misc); env != nil {
		view.Blocks = append(view.Blocks, ItemCompareBlock{Compare: flaggedOutputs(env)})
	}
	return view, nil
}

// flaggedOutputs is compareOutputs with each actor's true output entries, which the stat list's condFuncs read, as 1.
func flaggedOutputs(env *calculator2.Environment) CompareOutputs {
	out := compareOutputs(env)
	out.Output = withFlags(out.Output, env.Player.OutputFlags)
	if main := env.Player.MainSkill; main != nil && main.Minion != nil {
		out.Minion = withFlags(out.Minion, main.Minion.OutputFlags)
	}
	return out
}

func withFlags(output map[string]float64, flags map[string]bool) map[string]float64 {
	out := maps.Clone(output)
	for k, v := range flags {
		if v {
			out[k] = 1
		}
	}
	return out
}
