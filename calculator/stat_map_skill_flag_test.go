package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestStatMapSkillFlagIsEmitted is the producer half: PoB's `skillFlag` on a statMap entry, which the emitter dropped SILENTLY.
func TestStatMapSkillFlagIsEmitted(t *testing.T) {
	for _, c := range []struct{ stat, want string }{
		{"infinite_minion_duration", "permanentMinion"},
		{"skill_can_fire_arrows", "arrow"},
	} {
		entry, ok := raw.SkillStatMap[c.stat]
		if !ok {
			t.Errorf("%s is absent from the global stat map", c.stat)
			continue
		}
		if entry.SkillFlag != c.want {
			t.Errorf("%s SkillFlag = %q, want %q — SkillStatMap.lua declares it and the emitter "+
				"dropped every string scalar", c.stat, entry.SkillFlag, c.want)
		}
	}

	flagged := 0
	for _, entry := range raw.SkillStatMap {
		if entry.SkillFlag != "" {
			flagged++
		}
	}
	if flagged != 2 {
		t.Errorf("%d global stat-map entries carry a SkillFlag; the pin has exactly 2 "+
			"(SkillStatMap.lua:25 and :1342)", flagged)
	}

	for _, stat := range []string{"infinite_minion_duration", "skill_can_fire_arrows"} {
		if entry, ok := raw.SkillStatMap[stat]; ok && len(entry.Mods) != 0 {
			t.Errorf("%s carries %d mods; PoB's entry is flag-only", stat, len(entry.Mods))
		}
	}
}

// TestPermanentMinionFlagReachesTheSkill is the consumer half.
func TestPermanentMinionFlagReachesTheSkill(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	build := loadCorpusBuild(t, "1292-luminary-summon-reaper-of-revenants.xml")
	env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)

	main := env.Player.MainSkill
	if main == nil {
		t.Fatal("REFUSING: the build produced no main skill, so nothing was exercised")
	}
	id := ""
	if main.ActiveEffect != nil && main.ActiveEffect.GrantedEffect != nil {
		id = main.ActiveEffect.GrantedEffect.ID()
	}

	if flags, ok := raw.SkillBaseFlags[id]; ok {
		for _, f := range flags {
			if f == "permanentMinion" {
				t.Fatalf("CONTROL IS VACUOUS: main skill %q already carries permanentMinion in "+
					"baseFlags, so this build cannot show the statMap producer doing anything", id)
			}
		}
	}
	if !main.SkillFlags[SkillFlagMinion] {
		t.Fatalf("CONTROL IS VACUOUS: main skill %q is not a minion skill; pick a fixture whose "+
			"main skill declares infinite_minion_duration", id)
	}

	if !main.SkillFlags[SkillFlagPermanentMinion] {
		t.Errorf("main skill %q does not carry SkillFlagPermanentMinion. It declares "+
			"infinite_minion_duration and does not get the flag from baseFlags, so "+
			"CalcActiveSkill.lua:341-347's stat-map pass is the only producer — and "+
			"calc_perform.go:1600/:1637/:4809 all read it", id)
	}
}
