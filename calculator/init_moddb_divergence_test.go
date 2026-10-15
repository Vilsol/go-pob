package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestInitModDBMatchesPoBBaseMods: initModDB's name set was diffed against CalcSetup.lua:19-110 entry by entry.
//
//pob:block:src/Modules/CalcSetup.lua:20-116
func TestInitModDBMatchesPoBBaseMods(t *testing.T) {
	db := moddb.NewModDB()
	initModDB(&Environment{}, db)

	baseFlagWithCondition := func(name, condition string) bool {
		for _, m := range db.Mods[name] {
			if m.Type() != mod.TypeFlag || m.GetSource() != "Base" {
				continue
			}
			for _, tag := range m.Tags() {
				c, ok := tag.(*mod.ConditionTag)
				if !ok || c.Negative {
					continue
				}
				for _, v := range c.VarList {
					if v == condition {
						return true
					}
				}
			}
		}
		return false
	}

	t.Run("control: Burning IS implied by Ignited", func(t *testing.T) {
		if !baseFlagWithCondition("Condition:Burning", "Ignited") {
			t.Fatalf("REFUSING: the Burning-from-Ignited implication PoB DOES have reads absent, " +
				"so this probe cannot distinguish an absent mod from a broken matcher")
		}
	})

	t.Run("Chilled is NOT implied by Frozen", func(t *testing.T) {
		if baseFlagWithCondition("Condition:Chilled", "Frozen") {
			t.Errorf("Condition:Chilled is set from Frozen. PoB has no such implication at any " +
				"site: CalcSetup.lua:72-73 seeds only Burning-from-Ignited and " +
				"Poisoned-from-PoisonStack, and conditionFrozen (ConfigOptions.lua:1193-1195) " +
				"writes Condition:Frozen alone")
		}
	})

	t.Run("no base Virulence cap", func(t *testing.T) {
		if got := db.Sum(mod.TypeBase, nil, "Multiplier:VirulenceStacksMax"); got != 0 {
			t.Errorf("Multiplier:VirulenceStacksMax base = %v, want 0 — PoB seeds none", got)
		}
	})

	t.Run("brine charges", func(t *testing.T) {
		if _, ok := db.Mods["BrineChargesMax"]; !ok {
			t.Errorf("BrineChargesMax is absent (CalcSetup.lua:40)")
		}
		for name, want := range map[string]float64{
			"PhysicalDamageGainAsCold":      data.PhysicalDamagePercentToAddAsColdPerBrineCharge,
			"PhysicalDamageGainAsLightning": data.PhysicalDamagePercentToAddAsLightningPerBrineCharge,
		} {
			found := false
			for _, m := range db.Mods[name] {
				if m.GetSource() != "Base" || m.Type() != mod.TypeBase {
					continue
				}
				for _, tag := range m.Tags() {
					mt, ok := tag.(*mod.MultiplierTag)
					if !ok {
						continue
					}
					for _, v := range mt.VariableList {
						if v != "BrineCharge" {
							continue
						}
						found = true
						if m.Value().Float() != want {
							t.Errorf("%s per BrineCharge = %v, want %v", name, m.Value().Float(), want)
						}
					}
				}
			}
			if !found {
				t.Errorf("%s per BrineCharge is absent (CalcSetup.lua:41-42)", name)
			}
		}
	})
}
