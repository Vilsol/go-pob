package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

// TestSoulGainPreventionDurationRows pins ModParser.lua:2280 and :2281.
func TestSoulGainPreventionDurationRows(t *testing.T) {
	requireVaalInc := func(t *testing.T, e *ModCacheEntry) mod.Mod {
		t.Helper()
		if e == nil || e.Extra != "" || len(e.ModList) != 1 {
			t.Fatalf("got %v", e)
		}
		m := e.ModList[0]
		if m.Name() != "SoulGainPreventionDuration" || m.Type() != mod.TypeIncrease {
			t.Fatalf("got %s %s, want SoulGainPreventionDuration INC", m.Name(), m.Type())
		}
		if v := m.Value().Float(); v != -40 {
			t.Errorf("value = %v, want -40", v)
		}
		var st *mod.SkillTypeTag
		for _, tg := range m.Tags() {
			if s, ok := tg.(*mod.SkillTypeTag); ok {
				st = s
			}
		}
		if st == nil || len(st.SkillTypeList) != 1 || st.SkillTypeList[0] != string(data.SkillTypeVaal) {
			t.Errorf("SkillType tag = %+v, want Vaal", st)
		}
		return m
	}

	t.Run("vaal skills used during effect", func(t *testing.T) {
		m := requireVaalInc(t, ParseMod("Vaal Skills used during effect have 40% reduced Soul Gain Prevention Duration", false))
		var ct *mod.ConditionTag
		for _, tg := range m.Tags() {
			if c, ok := tg.(*mod.ConditionTag); ok {
				ct = c
			}
		}
		if ct == nil || len(ct.VarList) != 1 || ct.VarList[0] != "UsingFlask" {
			t.Errorf("Condition tag = %+v, want UsingFlask", ct)
		}
	})

	t.Run("vaal volcanic fissure and vaal molten strike", func(t *testing.T) {
		m := requireVaalInc(t, ParseMod("Vaal Volcanic Fissure and Vaal Molten Strike have 40% reduced Soul Gain Prevention Duration", false))
		var sn *mod.SkillNameTag
		for _, tg := range m.Tags() {
			if n, ok := tg.(*mod.SkillNameTag); ok {
				sn = n
			}
		}
		if sn == nil || len(sn.SkillNameList) != 2 ||
			sn.SkillNameList[0] != "Volcanic Fissure" || sn.SkillNameList[1] != "Molten Strike" {
			t.Fatalf("SkillName tag = %+v, want [Volcanic Fissure Molten Strike]", sn)
		}
		if !sn.IncludeTransfigured {
			t.Error("IncludeTransfigured = false, want true")
		}
	})
}
