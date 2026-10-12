package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestWarcriesCostLifeRowParses pins ModParser.lua:2279.
func TestWarcriesCostLifeRowParses(t *testing.T) {
	e := ParseMod("Warcries Cost +15% of Life", false)
	if e.Extra != "" || len(e.ModList) != 1 {
		t.Fatalf("remainder %q, %d mods", e.Extra, len(e.ModList))
	}
	m := e.ModList[0]
	if m.Name() != "LifeCostBase" || m.Type() != mod.TypeBase || m.KeywordFlags() != mod.KeywordFlagWarcry {
		t.Fatalf("got %s %s kw=%v, want LifeCostBase BASE kw=Warcry", m.Name(), m.Type(), m.KeywordFlags())
	}
	if len(m.Tags()) != 1 {
		t.Fatalf("got %d tags, want 1 PercentStat", len(m.Tags()))
	}
	if ps, ok := m.Tags()[0].(*mod.PercentStatTag); !ok || ps.Stat != "Life" || ps.Percent != 15 || !ps.TagFloor {
		t.Errorf("tag %#v, want PercentStat Life 15 floor", m.Tags()[0])
	}

	n := ParseMod("Warcries Cost no Mana", false)
	if n.Extra != "" || len(n.ModList) != 1 || n.ModList[0].Name() != "ManaCost" {
		t.Errorf("neighbour: remainder %q, mods %v", n.Extra, n.ModList)
	}
}

// TestWarcryLifeCostReachesTheCost is the end-to-end on the build that exposed it.
func TestWarcryLifeCostReachesTheCost(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "14675-ladder-sementenegra2_0161.COTA_CryCry.xml")
	if got := env.Player.Output["LifeCost"]; got < 1956.8*0.995 || got > 1956.8*1.005 {
		t.Errorf("LifeCost = %v, PoB's oracle has 1956.8", got)
	}
}
