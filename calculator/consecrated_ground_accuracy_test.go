package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestConsecratedGroundAlsoAccuracy covers ModParser.lua:4121 and CalcPerform.lua:774.
func TestConsecratedGroundAlsoAccuracy(t *testing.T) {
	requireGameData(t, "3.29")

	e := ParseMod("Consecrated Ground you create grants 30% increased Accuracy Rating to you and Allies", false)
	if e == nil || len(e.ModList) != 1 {
		t.Fatalf("REFUSING: the line did not parse to exactly one mod: %+v", e)
	}
	if extra := strings.TrimSpace(modExtra(e)); extra != "" {
		t.Fatalf("leftover Extra %q — one unmatched phrase discards the whole line", extra)
	}

	outer := e.ModList[0]
	if outer.Name() != "ExtraAura" {
		t.Fatalf("outer mod is %q, want ExtraAura — a plain mod would not reach allies", outer.Name())
	}
	aura, ok := outer.Value().List().(mod.ExtraAura)
	if !ok {
		t.Fatalf("ExtraAura payload is %T, want mod.ExtraAura", outer.Value().List())
	}
	inner := aura.Mod
	if inner.Name() != "ConsecratedGroundAlsoAccuracy" {
		t.Fatalf("nested mod is %q, want ConsecratedGroundAlsoAccuracy — CalcPerform.lua:774 reads "+
			"that exact name and nothing else consumes it", inner.Name())
	}
	if inner.Type() != mod.TypeIncrease || inner.Value().Float() != 30 {
		t.Fatalf("nested mod is %v %v, want INC 30", inner.Type(), inner.Value().Float())
	}
	var sawCondition bool
	for _, tag := range inner.Tags() {
		c, ok := tag.(*mod.ConditionTag)
		if !ok {
			continue
		}
		for _, v := range c.VarList {
			if v == "OnConsecratedGround" {
				sawCondition = true
			}
		}
	}
	if !sawCondition {
		t.Fatalf("nested mod carries no OnConsecratedGround condition; it would apply everywhere")
	}
}
