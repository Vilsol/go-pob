package calculator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestRadiusResistConversionLife pins ModParser.lua:6138 on 0591-luminary-precision.
func TestRadiusResistConversionLife(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	const name = "0591-luminary-precision.xml"
	rec, ok := oracle[name]
	if !ok {
		t.Fatalf("REFUSING: %s is not in the oracle blob", name)
	}
	want := rec.Output["Life"]
	if want == 0 {
		t.Fatalf("REFUSING: %s publishes no Life; the fixture has moved", name)
	}

	file, err := os.ReadFile(filepath.Join(corpusDir, name))
	if err != nil {
		t.Fatalf("%v", err)
	}
	build, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatalf("%v", err)
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())

	got := env.Player.Output["Life"]
	if got != want {
		t.Errorf("%s: Life = %v, want %v (PoB)", name, got, want)
	}
}

// TestRadiusSimpleConvFamilyParses walks ModParser.lua:6102-6107 and :6111-6143 at percentages this port never hardcoded.
func TestRadiusSimpleConvFamilyParses(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("Increases and Reductions to Life in Radius are Transformed to apply to Energy Shield", false); e == nil ||
		len(e.ModList) == 0 || strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: a non-parameterised radius row does not parse; the jewel-func path is " +
			"broken generally and this test cannot attribute a failure")
	}

	for _, line := range []string{
		"Passives granting Lightning Resistance or all Elemental Resistances in Radius also grant Chance to Block Spell Damage at 50% of its value",
		"Passives granting Lightning Resistance or all Elemental Resistances in Radius also grant Chance to Block Spells at 35% of its value",
		"Passives granting Lightning Resistance or all Elemental Resistances in Radius also grant increased Maximum Energy Shield at 75% of its value",
		"Passives granting Lightning Resistance or all Elemental Resistances in Radius also grant Lightning Damage converted to Chaos Damage at 30% of its value",
		"Passives granting Cold Resistance or all Elemental Resistances in Radius also grant Chance to Dodge Attack Hits at 35% of its value",
		"Passives granting Cold Resistance or all Elemental Resistances in Radius also grant Chance to Dodge Attacks at 35% of its value",
		"Passives granting Cold Resistance or all Elemental Resistances in Radius also grant Chance to Suppress Spell Damage at 70% of its value",
		"Passives granting Cold Resistance or all Elemental Resistances in Radius also grant increased Maximum Mana at 40% of its value",
		"Passives granting Cold Resistance or all Elemental Resistances in Radius also grant Cold Damage converted to Chaos Damage at 30% of its value",
		"Passives granting Fire Resistance or all Elemental Resistances in Radius also grant Chance to Block Attack Damage at 50% of its value",
		"Passives granting Fire Resistance or all Elemental Resistances in Radius also grant Chance to Block at 35% of its value",
		"Passives granting Fire Resistance or all Elemental Resistances in Radius also grant increased Maximum Life at 50% of its value",
		"Passives granting Fire Resistance or all Elemental Resistances in Radius also grant increased Maximum Life at 75% of its value",
		"Passives granting Fire Resistance or all Elemental Resistances in Radius also grant Fire Damage converted to Chaos Damage at 30% of its value",
		"Increases and Reductions to Energy Shield in Radius are Transformed to apply to Armour at 200% of their value",
		"Increases and Reductions to Life in Radius are Transformed to apply to Mana at 100% of their value",
	} {
		e := ParseMod(line, false)
		if e == nil {
			t.Errorf("no entry: %s", line)
			continue
		}
		if extra := strings.TrimSpace(modExtra(e)); extra != "" {
			t.Errorf("leftover %q (line is DROPPED WHOLE): %s", extra, line)
			continue
		}
		var got []string
		for _, m := range e.ModList {
			got = append(got, m.Name())
		}
		if !contains(got, "JewelFunc") {
			t.Errorf("produced %v, want a JewelFunc mod: %s", got, line)
		}
	}
}

// TestJewelFuncPatternsDoNotShadowExactRows is the standing double-apply detector.
func TestJewelFuncPatternsDoNotShadowExactRows(t *testing.T) {
	if len(jewelFuncPatterns) == 0 || len(jewelFuncList) == 0 {
		t.Fatalf("REFUSING: %d patterns / %d exact rows — the tables did not initialise",
			len(jewelFuncPatterns), len(jewelFuncList))
	}
	for line := range jewelFuncList {
		for _, p := range jewelFuncPatterns {
			if p.re.MatchString(line) {
				t.Errorf("exact row %q is shadowed by pattern %q", line, p.re.String())
			}
		}
	}
}

// TestSimpleConvSkipsTattooAttributes pins ModParser.lua:25's `not (node.isTattoo and attributes[src])`.
func TestSimpleConvSkipsTattooAttributes(t *testing.T) {
	build := func(isTattoo bool, name string, modType mod.Type, value float64) (*radiusJewelNode, *moddb.ModList) {
		src := moddb.NewModList()
		src.AddMod(mod.NewFloat(name, modType, value).Source("Tree:1"))
		return &radiusJewelNode{id: "1", nodeType: "Normal", modList: src, isTattoo: isTattoo}, moddb.NewModList()
	}
	sum := func(l *moddb.ModList, name string, modType mod.Type) float64 {
		return l.Sum(modType, nil, name)
	}

	node, out := build(false, "Str", mod.TypeBase, 20)
	getSimpleConv([]string{"Str"}, "Dex", mod.TypeBase, true)(node, out, &radiusJewelData{})
	if got := sum(out, "Dex", mod.TypeBase); got != 20 {
		t.Fatalf("REFUSING: a non-tattoo node converts Str %v to Dex, want 20 — the func did not run", got)
	}

	node, out = build(true, "Str", mod.TypeBase, 20)
	getSimpleConv([]string{"Str"}, "Dex", mod.TypeBase, true)(node, out, &radiusJewelData{})
	if got := sum(out, "Dex", mod.TypeBase); got != 0 {
		t.Errorf("tattoo node converted Str to Dex %v, want 0 (ModParser.lua:25)", got)
	}
	if got := sum(out, "Str", mod.TypeBase); got != 0 {
		t.Errorf("tattoo node had Str removed by %v, want 0 (the `remove` arm must not run either)", got)
	}

	node, out = build(true, "FireResist", mod.TypeBase, 15)
	getSimpleConvSrc([]string{"FireResist", "ElementalResist"}, "Life", mod.TypeIncrease, false, mod.TypeBase, 0.5)(node, out, &radiusJewelData{})
	if got := sum(out, "Life", mod.TypeIncrease); got != 7 {
		t.Errorf("tattoo node converted FireResist 15 to Life INC %v, want 7 — the guard is "+
			"Str/Dex/Int only and must not reach the pool rows", got)
	}
}
