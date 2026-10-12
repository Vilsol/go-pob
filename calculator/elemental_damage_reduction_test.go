package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestEnduranceChargesGiveElementalDamageReductionNotResistance pins CalcSetup.lua:515-516.
//
//	modDB:NewMod("PhysicalDamageReduction", "BASE", ..., "Base", { Multiplier EnduranceCharge })
//	modDB:NewMod("ElementalDamageReduction", "BASE", ..., "Base", { Multiplier EnduranceCharge })
func TestEnduranceChargesGiveElementalDamageReductionNotResistance(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0011-saboteur-cyclone.xml")

	found := false
	for _, r := range env.ModDB.Tabulate(mod.TypeBase, nil, "ElementalDamageReduction") {
		if r.Mod != nil && r.Mod.Name() == "ElementalDamageReduction" {
			found = true
		}
	}
	if !found {
		t.Error("no ElementalDamageReduction mod on the player; endurance charges feed it at " +
			"CalcSetup.lua:516 and Base<elem>DamageReduction has nothing else to sum")
	}

	for _, r := range env.ModDB.Tabulate(mod.TypeBase, nil, "ElementalResist") {
		if r.Mod == nil || r.Mod.Name() != "ElementalResist" {
			continue
		}
		for _, tag := range r.Mod.Tags() {
			mult, ok := tag.(*mod.MultiplierTag)
			if !ok {
				continue
			}
			for _, v := range mult.VariableList {
				if v == "EnduranceCharge" {
					t.Errorf("an ElementalResist mod is scaled by EnduranceCharge; PoB's only " +
						"`ElementalResist BASE` in CalcSetup is the Bandit reward (:551), so " +
						"this grants resistance no character has")
				}
			}
		}
	}
}

// TestBaseElementalDamageReductionMatchesPoB grades the output the fix above feeds.
func TestBaseElementalDamageReductionMatchesPoB(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	stats := []string{
		"BaseFireDamageReduction", "BaseColdDamageReduction", "BaseLightningDamageReduction",
		"BaseFireDamageReductionWhenHit", "BaseColdDamageReductionWhenHit", "BaseLightningDamageReductionWhenHit",
		"BasePhysicalDamageReduction", "BaseChaosDamageReduction",
	}

	seen, agreed := 0, 0
	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		var want map[string]float64
		for _, s := range stats {
			if v, ok := rec.Output[s]; ok && v != 0 {
				if want == nil {
					want = map[string]float64{}
				}
				want[s] = v
			}
		}
		if want == nil {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		for s, v := range want {
			seen++
			if env.Player.Output[s] == v {
				agreed++
			}
		}
		if seen >= 200 {
			break
		}
	}

	if seen == 0 {
		t.Fatal("REFUSING: no graded build reports a nonzero Base<type>DamageReduction; this test " +
			"no longer exercises the mechanism it was written for")
	}
	t.Logf("Base<type>DamageReduction: %d of %d sampled (build, stat) pairs match PoB exactly", agreed, seen)

	const agreementFloor = 201
	if agreed < agreementFloor {
		t.Errorf("Base<type>DamageReduction matches PoB on %d of %d pairs, below the floor of %d "+
			"measured when CalcDefence.lua:1499's ElementalDamageReduction sum landed", agreed, seen, agreementFloor)
	}
}
