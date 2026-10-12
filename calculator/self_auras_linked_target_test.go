package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

// TestSelfAurasAffectYouAndLinkedTargetSuppressesTheEnemyArm pins ModParser.lua:4328 against its consumer at CalcPerform.lua:2451.
func TestSelfAurasAffectYouAndLinkedTargetSuppressesTheEnemyArm(t *testing.T) {
	t.Run("the flag parses", func(t *testing.T) {
		const line = "Non-Curse Auras from your Skills only apply to you and Linked targets"
		entry := ParseMod(line, false)
		if entry.Extra != "" {
			t.Fatalf("ModParser.lua:4328 — %q left %q unparsed", line, entry.Extra)
		}
		if len(entry.ModList) != 1 {
			t.Fatalf("ModParser.lua:4328 — %q produced %d mods, want 1", line, len(entry.ModList))
		}
		m := entry.ModList[0]
		if m.Name() != "SelfAurasAffectYouAndLinkedTarget" {
			t.Fatalf("mod name = %s, want SelfAurasAffectYouAndLinkedTarget", m.Name())
		}
		if len(m.Tags()) != 2 {
			t.Fatalf("got %d tags, want 2 (SkillType Aura, SkillType AppliesCurse neg)", len(m.Tags()))
		}
		for i, want := range []struct {
			skillType string
			neg       bool
		}{
			{string(data.SkillTypeAura), false},
			{string(data.SkillTypeAppliesCurse), true},
		} {
			st, ok := m.Tags()[i].(*mod.SkillTypeTag)
			if !ok {
				t.Fatalf("tag %d is %#v, want a SkillType", i, m.Tags()[i])
			}
			if len(st.SkillTypeList) != 1 || st.SkillTypeList[0] != want.skillType || st.Negative != want.neg {
				t.Fatalf("tag %d = %v neg=%v, want [%s] neg=%v", i, st.SkillTypeList, st.Negative, want.skillType, want.neg)
			}
		}
	})

	t.Run("the aura stops at the enemy and not at you", func(t *testing.T) {
		if os.Getenv("CORPUS") != "true" {
			t.SkipNow()
		}
		requireGameData(t, "3.29")

		const name = "0554-luminary-stormblast-mine.xml"
		xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), name))
		if err != nil {
			t.Fatal(err)
		}
		build, err := builds.ParseBuild(xml)
		if err != nil {
			t.Fatal(err)
		}
		env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)
		if env == nil || env.Player == nil || env.Enemy == nil {
			t.Fatal("no MAIN env")
		}

		if n := len(env.Enemy.ModDB.Mods["CritChance"]); n != 0 {
			t.Errorf("enemy carries %d CritChance mods, want 0 — the aura reached the enemy", n)
		}

		found := false
		for _, m := range env.ModDB.Mods["CritChance"] {
			if m.GetSource() == mod.Source("Skill:Precision") {
				found = true
			}
		}
		if !found {
			t.Error("player carries no CritChance from Skill:Precision — the aura was suppressed outright")
		}

		for _, tc := range []struct {
			key  string
			want float64
			role string
		}{
			{"EnemyCritChance", 5, "the subject: the enemy's bare default, with no aura on it"},
			{"EnemyCritEffect", 1.015, "one rung down"},
			{"FireEnemyDamage", 2274.61, "the taken-damage chain the rung feeds"},
		} {
			got := env.Player.Output[tc.key]
			if math.Abs(got-tc.want) > math.Abs(tc.want)*0.005 {
				t.Errorf("%s = %g, want %g (%s)", tc.key, got, tc.want, tc.role)
			}
		}
	})
}
