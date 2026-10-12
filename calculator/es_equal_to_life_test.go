package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestMaximumEnergyShieldEqualToLifeParses pins ModParser.lua:5868-5870, the Ghostwrithe line.
func TestMaximumEnergyShieldEqualToLifeParses(t *testing.T) {
	e := ParseMod("Your Maximum Energy Shield is Equal to 39% of Your Maximum Life", false)
	if e == nil || len(e.ModList) != 1 {
		t.Fatalf("ParseMod produced %v, want 1 mod", e)
	}
	if e.Extra != "" {
		t.Fatalf("leftover Extra %q — processModLine discards the whole line", e.Extra)
	}
	m := e.ModList[0]
	if m.Name() != "EnergyShield" || m.Type() != mod.TypeOverride {
		t.Fatalf("mod = %s/%s, want EnergyShield/OVERRIDE", m.Name(), m.Type())
	}
	tags := m.Tags()
	if len(tags) != 1 {
		t.Fatalf("tags = %v, want one PercentStat", tags)
	}
	ps, ok := tags[0].(*mod.PercentStatTag)
	if !ok {
		t.Fatalf("tag = %T, want *mod.PercentStatTag", tags[0])
	}
	if ps.Stat != "Life" || ps.Percent != 39 {
		t.Fatalf("PercentStat = %s/%v, want Life/39", ps.Stat, ps.Percent)
	}

	if e := ParseMod("Maximum Energy Shield is Equal to 50% of Maximum Life", false); e == nil || len(e.ModList) != 1 || e.Extra != "" {
		t.Errorf("the prefix-less spelling must parse whole; got %v", e)
	}
}
