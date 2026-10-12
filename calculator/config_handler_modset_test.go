package calculator

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

func emittedModSet(t *testing.T, name string, val interface{}) map[string]int {
	t.Helper()
	h, ok := configurations[name]
	if !ok {
		t.Fatalf("REFUSING: no handler named %q — an absent handler and an empty mod set are "+
			"different findings and this must not report them the same way", name)
	}
	modList, enemyModList := moddb.NewModList(), moddb.NewModList()
	h(val, modList, enemyModList)

	out := map[string]int{}
	var add func(prefix string, m mod.Mod)
	add = func(prefix string, m mod.Mod) {
		out[fmt.Sprintf("%s%s|%s", prefix, m.Name(), m.Type())]++
		if m.Value().Type() == mod.ModValueMultiTypeList {
			switch inner := m.Value().List().(type) {
			case mod.MinionModifier:
				add(prefix, inner.Mod)
			case mod.ExtraSkillMod:
				add(prefix, inner.Mod)
			}
		}
	}
	for _, m := range modList.Mods() {
		add("", m)
	}
	for _, m := range enemyModList.Mods() {
		add("enemy:", m)
	}
	return out
}

func assertModSet(t *testing.T, name string, val interface{}, want ...string) {
	t.Helper()
	got := emittedModSet(t, name, val)
	wantSet := map[string]int{}
	for _, w := range want {
		wantSet[w]++
	}
	if len(got) == 0 {
		t.Fatalf("%s: emitted NOTHING for val=%v — the handler's own guard rejected the fixture, "+
			"so this asserts nothing about the mod set", name, val)
	}
	var missing, extra []string
	for k, n := range wantSet {
		if got[k] < n {
			missing = append(missing, fmt.Sprintf("%s x%d", k, n-got[k]))
		}
	}
	for k, n := range got {
		if wantSet[k] < n {
			extra = append(extra, fmt.Sprintf("%s x%d", k, n-wantSet[k]))
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) > 0 || len(extra) > 0 {
		t.Errorf("%s mod set diverges from the pin:\n  PoB-only: %s\n  ours-only: %s",
			name, strings.Join(missing, ", "), strings.Join(extra, ", "))
	}
}

func assertSoleConditionTag(t *testing.T, name string, val interface{}, wantVar string) {
	t.Helper()
	h, ok := configurations[name]
	if !ok {
		t.Fatalf("REFUSING: no handler named %q", name)
	}
	modList, enemyModList := moddb.NewModList(), moddb.NewModList()
	h(val, modList, enemyModList)

	mods := append(append([]mod.Mod{}, modList.Mods()...), enemyModList.Mods()...)
	if len(mods) != 1 {
		t.Fatalf("REFUSING: %s emitted %d mods, not 1 — this assertion cannot say which one "+
			"carries the tag it is checking", name, len(mods))
	}
	var got []string
	for _, tg := range mods[0].Tags() {
		c, ok := tg.(*mod.ConditionTag)
		if !ok {
			continue
		}
		got = append(got, c.VarList...)
	}
	if len(got) != 1 || got[0] != wantVar {
		t.Errorf("%s: Condition tag vars = %v, ConfigOptions.lua has exactly [%s]", name, got, wantVar)
	}
}

// TestConfigHandlerModSetsMatchThePin pins the name+type multiset each handler emits against the pinned ConfigOptions.lua apply function.
func TestConfigHandlerModSetsMatchThePin(t *testing.T) {
	t.Run("buffUnholyMight", func(t *testing.T) {
		assertModSet(t, "buffUnholyMight", true,
			"Condition:UnholyMight|FLAG", "Condition:CanWither|FLAG")
	})
	t.Run("minionbuffUnholyMight", func(t *testing.T) {
		assertModSet(t, "minionbuffUnholyMight", true,
			"MinionModifier|LIST", "Condition:UnholyMight|FLAG",
			"MinionModifier|LIST", "Condition:CanWither|FLAG")
	})
	t.Run("multiplierChilledByYouSeconds", func(t *testing.T) {
		assertModSet(t, "multiplierChilledByYouSeconds", float64(3),
			"enemy:Multiplier:ChilledByYouSeconds|BASE", "enemy:Condition:ChilledByYou|FLAG")
	})
	t.Run("conditionEnemyOnProfaneGround", func(t *testing.T) {
		assertModSet(t, "conditionEnemyOnProfaneGround", true,
			"enemy:Condition:OnProfaneGround|FLAG",
			"enemy:CurseEffectOnSelf|INC",
			"enemy:SelfCritChance|INC")
	})
	t.Run("aspectOfTheSpiderWebStacks", func(t *testing.T) {
		assertModSet(t, "aspectOfTheSpiderWebStacks", float64(2),
			"ExtraSkillMod|LIST", "Multiplier:SpiderWebApplyStack|BASE",
			"Condition:AspectOfTheSpiderActive|FLAG")
	})
	t.Run("raiseSpidersSpiderCount", func(t *testing.T) {
		assertModSet(t, "raiseSpidersSpiderCount", float64(7),
			"Multiplier:RaisedSpiderConfig|BASE", "Multiplier:RaisedSpider|BASE")
	})
	t.Run("configSpectralWolfCount", func(t *testing.T) {
		assertModSet(t, "configSpectralWolfCount", float64(4),
			"Multiplier:SpectralWolfConfig|BASE", "Multiplier:SpectralWolfCount|BASE")
	})
	t.Run("BrandsInLastQuarter", func(t *testing.T) {
		assertModSet(t, "BrandsInLastQuarter", true,
			"Condition:BrandLastQuarter|FLAG", "Condition:BrandLastHalf|FLAG")
	})
	t.Run("conditionShocked", func(t *testing.T) {
		assertModSet(t, "conditionShocked", true, "Condition:Shocked|FLAG")
	})
	t.Run("conditionPlayerShockEffect", func(t *testing.T) {
		assertModSet(t, "conditionPlayerShockEffect", float64(20), "ShockVal|OVERRIDE")
	})
	for _, tc := range []struct {
		handler string
		val     interface{}
		want    string
		tagVar  string
	}{
		{"conditionPoisonedEnemyRecently", true, "Condition:PoisonedEnemyRecently|FLAG", "Combat"},
		{"conditionCausedBleedingRecently", true, "Condition:CausedBleedingRecently|FLAG", "Combat"},
		{"conditionKilledPoisonedLast2Seconds", true, "Condition:KilledPoisonedLast2Seconds|FLAG", "Combat"},
		{"enemyMultiplierEnemyPresenceSeconds", float64(10), "enemy:Multiplier:EnemyPresenceSeconds|BASE", "Effective"},
	} {
		t.Run(tc.handler, func(t *testing.T) {
			assertModSet(t, tc.handler, tc.val, tc.want)
			assertSoleConditionTag(t, tc.handler, tc.val, tc.tagVar)
			n := 0
			for _, name := range configOrder {
				if name == tc.handler {
					n++
				}
			}
			if n != 1 {
				t.Errorf("%s appears %d times in configOrder, want exactly 1", tc.handler, n)
			}
		})
	}
	for _, boss := range []string{"Boss", "Pinnacle", "Uber"} {
		t.Run("enemyIsBoss/"+boss, func(t *testing.T) {
			got := emittedModSet(t, "enemyIsBoss", boss)
			if got["Multiplier:EnemyPower|BASE"] != 1 {
				t.Errorf("enemyIsBoss=%s: Multiplier:EnemyPower BASE emitted %d times, want 1",
					boss, got["Multiplier:EnemyPower|BASE"])
			}
		})
	}
}

// TestDeliriumAndScorchValuesMatchThePin pins the two divergences a NAME-set diff cannot see.
func TestDeliriumAndScorchValuesMatchThePin(t *testing.T) {
	for val, want := range map[string]float64{
		"20Percent": -16, "40Percent": -32, "60Percent": -48, "80Percent": -64, "100Percent": -80,
	} {
		modList, enemyModList := moddb.NewModList(), moddb.NewModList()
		configurations["deliriousPercentage"](val, modList, enemyModList)
		got := 0.0
		found := false
		for _, m := range enemyModList.Mods() {
			if m.Name() == "DamageTaken" && m.Type() == mod.TypeMore {
				got, found = m.Value().Float(), true
			}
		}
		if !found {
			t.Fatalf("deliriousPercentage=%s emitted no DamageTaken MORE — the handler's own "+
				"string compare rejected the fixture, so this asserts nothing", val)
		}
		if got != want {
			t.Errorf("deliriousPercentage=%s: DamageTaken MORE = %g, ConfigOptions.lua has %g",
				val, got, want)
		}
	}
	modList, enemyModList := moddb.NewModList(), moddb.NewModList()
	configurations["conditionScorchedEffect"](float64(20), modList, enemyModList)
	for _, m := range enemyModList.Mods() {
		if m.Name() == "DesiredScorchVal" && m.GetSource() != "Scorch" {
			t.Errorf("DesiredScorchVal source = %q, want %q", m.GetSource(), "Scorch")
		}
	}
}
