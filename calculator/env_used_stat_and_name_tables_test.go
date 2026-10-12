package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestUsedTablesRecordModNamesTagTypesAndStats covers the four tables Calcs.lua:498-501 declares that this port never created.
func TestUsedTablesRecordModNamesTagTypesAndStats(t *testing.T) {
	newEnv := func() *Environment {
		env := &Environment{}
		env.initUsedTables()
		return env
	}
	has := func(t *testing.T, table map[string][]mod.Mod, key, label string) {
		t.Helper()
		if len(table[key]) == 0 {
			keys := make([]string, 0, len(table))
			for k := range table {
				keys = append(keys, k)
			}
			t.Errorf("%s: no entry for %q; table holds %v", label, key, keys)
		}
	}

	t.Run("mod name and tag types", func(t *testing.T) {
		env := newEnv()
		env.addModTagsUsed(true, mod.NewFloat("Damage", mod.TypeIncrease, 10).
			Tag(mod.Condition("Fortified")).Source("Item"))
		has(t, env.ModsUsed, "Damage", "ModsUsed")
		has(t, env.TagTypesUsed, "Condition", "TagTypesUsed")
		has(t, env.ConditionsUsed, "Fortified", "ConditionsUsed")
	})

	t.Run("a tagless mod still reaches ModsUsed", func(t *testing.T) {
		env := newEnv()
		env.addModTagsUsed(true, mod.NewFloat("Life", mod.TypeBase, 10).Source("Item"))
		has(t, env.ModsUsed, "Life", "ModsUsed")
	})

	t.Run("the IgnoreCond tag is counted and the tags after it are not", func(t *testing.T) {
		env := newEnv()
		env.addModTagsUsed(true, mod.NewFloat("Damage", mod.TypeIncrease, 10).
			Tag(mod.IgnoreCond()).Tag(mod.Condition("Ignited")).Source("Item"))
		has(t, env.TagTypesUsed, "IgnoreCond", "TagTypesUsed")
		if len(env.TagTypesUsed["Condition"]) != 0 {
			t.Errorf("the Condition tag after IgnoreCond was counted")
		}
		if len(env.ConditionsUsed["Ignited"]) != 0 {
			t.Errorf("the Condition tag after IgnoreCond reached ConditionsUsed")
		}
	})

	t.Run("PerStat files under the stat, for the right actor", func(t *testing.T) {
		env := newEnv()
		env.addModTagsUsed(true, mod.NewFloat("Damage", mod.TypeIncrease, 1).
			Tag(mod.PerStat(10, "Strength")).Source("Item"))
		env.addModTagsUsed(true, mod.NewFloat("Damage", mod.TypeIncrease, 1).
			Tag(mod.PerStat(10, "Armour").Actor("enemy")).Source("Item"))
		env.addModTagsUsed(true, mod.NewFloat("Damage", mod.TypeIncrease, 1).
			Tag(mod.StatThreshold("Life", 100)).Source("Item"))
		has(t, env.PerStatsUsed, "Strength", "PerStatsUsed")
		has(t, env.PerStatsUsed, "Life", "PerStatsUsed")
		has(t, env.EnemyPerStatsUsed, "Armour", "EnemyPerStatsUsed")
		if len(env.PerStatsUsed["Armour"]) != 0 {
			t.Errorf("an enemy-actor PerStat reached the player table")
		}
	})

	t.Run("a minion's unactored PerStat is dropped, like its Multiplier", func(t *testing.T) {
		env := newEnv()
		env.addModTagsUsed(false, mod.NewFloat("Damage", mod.TypeIncrease, 1).
			Tag(mod.PerStat(10, "Strength")).Source("Item"))
		if len(env.PerStatsUsed["Strength"]) != 0 {
			t.Errorf("a minion's unactored PerStat reached perStatsUsed; Calcs.lua:568-571 gates on the player")
		}
	})

	t.Run("the name implies the enemy ailment conditions", func(t *testing.T) {
		env := newEnv()
		env.addModTagsUsed(true, mod.NewFloat("FireDamage", mod.TypeIncrease, 10).Source("Item"))
		has(t, env.EnemyConditionsUsed, "Ignited", "EnemyConditionsUsed")
		has(t, env.EnemyConditionsUsed, "Burning", "EnemyConditionsUsed")

		env = newEnv()
		env.addModTagsUsed(true, mod.NewFloat("ColdDamage", mod.TypeIncrease, 10).Source("Item"))
		has(t, env.EnemyConditionsUsed, "Frozen", "EnemyConditionsUsed")
		if len(env.EnemyConditionsUsed["Ignited"]) != 0 {
			t.Errorf("ColdDamage implied Ignited")
		}
	})

	t.Run("a Base-sourced mod reaches none of them", func(t *testing.T) {
		env := newEnv()
		env.addModTagsUsed(true, mod.NewFloat("FireDamage", mod.TypeIncrease, 10).
			Tag(mod.PerStat(10, "Strength")).Source("Base"))
		for name, table := range map[string]map[string][]mod.Mod{
			"ModsUsed": env.ModsUsed, "TagTypesUsed": env.TagTypesUsed,
			"PerStatsUsed": env.PerStatsUsed, "EnemyConditionsUsed": env.EnemyConditionsUsed,
		} {
			if len(table) != 0 {
				t.Errorf("%s took a Base-sourced mod: %v", name, table)
			}
		}
	})
}
