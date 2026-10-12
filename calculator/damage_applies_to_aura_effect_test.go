package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
)

// TestDamageAppliesToAuraEffect pins ModParser.lua:3429-3433 against its consumer at CalcPerform.lua:1505-1529.
func TestDamageAppliesToAuraEffect(t *testing.T) {
	requireGameData(t, "3.29")

	t.Run("the row parses whole", func(t *testing.T) {
		const line = "Increases and Reductions to Chaos Damage also apply to Effect of Auras from Chaos Skills at 15% of their value, up to a maximum of 150%"
		e := ParseMod(line, false)
		if e.Extra != "" {
			t.Fatalf("unparsed remainder %q -- the WHOLE line is discarded", e.Extra)
		}
		want := map[string]struct {
			typ mod.Type
			val float64
		}{
			"ChaosDamageAppliesToChaosAuraEffect":         {mod.TypeFlag, 0},
			"ImprovedChaosDamageAppliesToChaosAuraEffect": {mod.TypeBase, 15},
			"ChaosDamageAppliesToChaosAuraEffectLimit":    {mod.TypeMAX, 150},
		}
		if len(e.ModList) != len(want) {
			t.Fatalf("got %d mods, want %d", len(e.ModList), len(want))
		}
		for _, m := range e.ModList {
			w, ok := want[m.Name()]
			if !ok {
				t.Fatalf("unexpected mod %s", m.Name())
			}
			if m.Type() != w.typ {
				t.Errorf("%s is %s, want %s", m.Name(), m.Type(), w.typ)
			}
			if w.typ != mod.TypeFlag && m.Value().Float() != w.val {
				t.Errorf("%s = %v, want %v", m.Name(), m.Value().Float(), w.val)
			}
			delete(want, m.Name())
		}
	})

	t.Run("a mismatched pair parses to names nothing reads", func(t *testing.T) {
		e := ParseMod("Increases and Reductions to Fire Damage also apply to Effect of Auras from Cold Skills at 10% of their value, up to a maximum of 50%", false)
		if e.Extra != "" {
			t.Fatalf("unparsed remainder %q", e.Extra)
		}
		found := false
		for _, m := range e.ModList {
			if m.Name() == "FireDamageAppliesToColdAuraEffect" {
				found = true
			}
		}
		if !found {
			t.Error("the flag name did not take the SECOND element -- both captures must reach it")
		}
	})

	if os.Getenv("CORPUS") != "true" {
		return
	}

	for _, tc := range []struct {
		build string
		kw    mod.KeywordFlag
		want  float64
		role  string
	}{
		{"0397-hierophant-bladefall-of-impaling", mod.KeywordFlagPhysical, 54, "uncapped: 22 conversions, well under 150"},
		{"6300-ladder-TheChekko_1992.Chekko", mod.KeywordFlagChaos, 150, "capped: the running total stops at the 150 limit"},
	} {
		t.Run(tc.build, func(t *testing.T) {
			xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), tc.build+".xml"))
			if err != nil {
				t.Fatal(err)
			}
			build, err := builds.ParseBuild(xml)
			if err != nil {
				t.Fatal(err)
			}
			env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)
			got := float64(0)
			for _, m := range env.ModDB.Mods["AuraEffect"] {
				if m.Type() == mod.TypeIncrease && m.KeywordFlags()&tc.kw != 0 {
					got += m.Value().Float()
				}
			}
			if got != tc.want {
				t.Errorf("converted AuraEffect = %v, want %v (%s)", got, tc.want, tc.role)
			}
		})
	}
}
