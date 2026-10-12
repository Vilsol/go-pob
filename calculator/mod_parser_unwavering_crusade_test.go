package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestUnwaveringCrusadeGrantsTheRelic pins ModParser.lua:3334.
//
//pob:src/Modules/ModParser.lua:3334
func TestUnwaveringCrusadeGrantsTheRelic(t *testing.T) {
	requireGameData(t, "3.29")

	const line = "25% chance to Trigger Level 20 Summon Elemental Relic when you or a nearby Ally Kill an Enemy, or Hit a Rare or Unique Enemy"
	e := ParseMod(line, false)
	if e == nil || len(e.ModList) == 0 {
		t.Fatalf("no mods at all — the line is DROPPED WHOLE: %s", line)
	}
	if extra := strings.TrimSpace(modExtra(e)); extra != "" {
		t.Fatalf("leftover %q — an unparsed remainder discards the WHOLE line: %s", extra, line)
	}
	if len(e.ModList) != 1 || e.ModList[0].Name() != "ExtraSkill" {
		got := make([]string, 0, len(e.ModList))
		for _, m := range e.ModList {
			got = append(got, m.Name())
		}
		t.Fatalf("mods %v, want exactly [ExtraSkill]", got)
	}
	es, ok := e.ModList[0].Value().List().(mod.ExtraSkill)
	if !ok {
		t.Fatalf("payload is %T, want mod.ExtraSkill", e.ModList[0].Value().List())
	}
	if es.SkillID != "SummonElementalRelic" {
		t.Fatalf("SkillID %q, want SummonElementalRelic — an empty id is cut at item.go and grants "+
			"nothing while reading as a fix", es.SkillID)
	}
	if es.Level != 20 {
		t.Fatalf("Level %d, want 20", es.Level)
	}
	if !es.Triggered {
		t.Fatalf("Triggered false, want true")
	}
	if es.NoSupports {
		t.Fatalf("NoSupports true, want false — upstream's options table sets only triggerChance")
	}

	nb := ParseMod("20% chance to Trigger Level 25 Summon Spectral Wolf on Critical Strike with this Weapon", false)
	if nb == nil || len(nb.ModList) != 2 {
		t.Fatalf("the :3333 neighbour no longer emits its ExtraSkill + ExtraSkillMod pair")
	}
}
