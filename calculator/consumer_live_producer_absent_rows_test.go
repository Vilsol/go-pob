package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestConsumerLiveProducerAbsentRows pins the parser rows recovered by running every DROPPED item mod line through the pinned PoB's LIVE grammar and keeping only the lines PoB parses WHOLE whose emitted mod name already has a reader in this tree.
func TestConsumerLiveProducerAbsentRows(t *testing.T) {
	type want struct {
		name         string
		modType      mod.Type
		value        any
		modFlags     mod.MFlag
		keywordFlags mod.KeywordFlag
		tags         int
	}

	for _, c := range []struct {
		line string
		want []want
	}{
		{line: "Unaffected by Ignite", want: []want{{name: "SelfIgniteEffect", modType: mod.TypeMore, value: -100.0}}},
		{line: "No Block Chance", want: []want{{name: "ArmourData", modType: mod.TypeList}}},
		{line: "Recover 10% of Life on Killing a Poisoned Enemy", want: []want{{name: "LifeOnKill", modType: mod.TypeBase, value: 1.0, tags: 2}}},
		{line: "Offering Skills have 20% reduced Duration", want: []want{{name: "Duration", modType: mod.TypeIncrease, value: -20.0, tags: 1}}},

		{line: "Unaffected by Ignite if 2 Warlord Items are Equipped", want: []want{{name: "SelfIgniteEffect", modType: mod.TypeMore, value: -100.0, tags: 1}}},
		{line: "Unaffected by Shock if 2 Crusader Items are Equipped", want: []want{{name: "SelfShockEffect", modType: mod.TypeMore, value: -100.0, tags: 1}}},
		{line: "Unaffected by Chill if 2 Redeemer Items are Equipped", want: []want{{name: "SelfChillEffect", modType: mod.TypeMore, value: -100.0, tags: 1}}},
		{line: "Unaffected by Poison if 2 Hunter Items are Equipped", want: []want{{name: "SelfPoisonEffect", modType: mod.TypeMore, value: -100.0, tags: 1}}},
		{line: "No Chance to Block", want: []want{{name: "ArmourData", modType: mod.TypeList}}},
		{line: "37% of Physical Damage is taken from Mana before Life", want: []want{{name: "PhysicalDamageTakenFromManaBeforeLife", modType: mod.TypeBase, value: 37.0}}},
		{line: "You are at Maximum Chance to Block Spell Damage if you have not Blocked Recently", want: []want{{name: "MaxSpellBlockIfNotBlockedRecently", modType: mod.TypeFlag, value: true, tags: 1}}},
		{line: "Stun Threshold is based on 500% of your Mana instead of Life", want: []want{
			{name: "StunThresholdBasedOnManaInsteadOfLife", modType: mod.TypeFlag, value: true},
			{name: "StunThresholdManaPercent", modType: mod.TypeBase, value: 500.0},
		}},
		{line: "Attacks cannot Hit you", want: []want{{name: "AlwaysEvade", modType: mod.TypeFlag, value: true}}},
		{line: "Modifiers to Ignite Duration on you apply to all Elemental Ailments", want: []want{{name: "IgniteDurationAppliesToElementalAilments", modType: mod.TypeFlag, value: true}}},
		{line: "Cannot deal Critical Strikes with Attacks", want: []want{
			{name: "NeverCrit", modType: mod.TypeFlag, value: true, modFlags: mod.MFlagAttack},
			{name: "Condition:NeverCrit", modType: mod.TypeFlag, value: true, modFlags: mod.MFlagAttack},
		}},
		{line: "Always Freeze, Shock and Ignite", want: []want{
			{name: "EnemyFreezeChance", modType: mod.TypeBase, value: 100.0},
			{name: "EnemyShockChance", modType: mod.TypeBase, value: 100.0},
			{name: "EnemyIgniteChance", modType: mod.TypeBase, value: 100.0},
		}},
		{line: "Melee Weapon Attacks have Culling Strike", want: []want{{name: "CullPercent", modType: mod.TypeMAX, value: 10.0, modFlags: mod.MFlagAttack | mod.MFlagWeaponMelee, tags: 1}}},
		{line: "You can apply an additional Curse while at maximum Power Charges", want: []want{{name: "EnemyCurseLimit", modType: mod.TypeBase, value: 1.0, tags: 1}}},
		{line: "Gain 23 Life per Cursed Enemy Hit with Attacks", want: []want{{name: "LifeOnHit", modType: mod.TypeBase, value: 23.0, modFlags: mod.MFlagAttack | mod.MFlagHit, tags: 1}}},
		{line: "Gain 11 Mana per Cursed Enemy Hit with Attacks", want: []want{{name: "ManaOnHit", modType: mod.TypeBase, value: 11.0, modFlags: mod.MFlagAttack | mod.MFlagHit, tags: 1}}},
		{line: "Deal no Physical or Elemental Damage", want: []want{
			{name: "DealNoPhysical", modType: mod.TypeFlag, value: true},
			{name: "DealNoCold", modType: mod.TypeFlag, value: true},
			{name: "DealNoFire", modType: mod.TypeFlag, value: true},
			{name: "DealNoLightning", modType: mod.TypeFlag, value: true},
		}},
		{line: "Your Melee Hits can't be Evaded while wielding a Sword", want: []want{{name: "CannotBeEvaded", modType: mod.TypeFlag, value: true, modFlags: mod.MFlagMelee | mod.MFlagHit, tags: 1}}},
		{line: "Chill Enemies as though dealing 84% more Damage", want: []want{{name: "ChillAsThoughDealing", modType: mod.TypeMore, value: 84.0}}},
		{line: "Projectiles Pierce all Burning Enemies", want: []want{{name: "PierceAllTargets", modType: mod.TypeFlag, value: true, tags: 1}}},
		{line: "Hits with this Weapon ignore Enemy Physical Damage Reduction", want: []want{{name: "ChanceToIgnoreEnemyPhysicalDamageReduction", modType: mod.TypeBase, value: 100.0, modFlags: mod.MFlagHit, tags: 2}}},
		{line: "Non-Instant Warcries ignore their Cooldown when Used", want: []want{{name: "CooldownRecovery", modType: mod.TypeOverride, value: 0.0, keywordFlags: mod.KeywordFlagWarcry, tags: 1}}},
		{line: "Nearby Enemy Monsters have at least 8% of Life Reserved", want: []want{{name: "EnemyModifier", modType: mod.TypeList}}},
		{line: "Minions Recover 10% of Life on Killing a Poisoned Enemy", want: []want{{name: "MinionModifier", modType: mod.TypeList}}},
		{line: "Minions have 60% chance to inflict Withered on Hit", want: []want{{name: "MinionModifier", modType: mod.TypeList}}},
		{line: "Raised Spectres fire 2 additional Projectiles", want: []want{{name: "MinionModifier", modType: mod.TypeList, tags: 1}}},
		{line: "Minions have 1% chance to deal Double Damage per Fortification on you", want: []want{{name: "MinionModifier", modType: mod.TypeList}}},
		{line: "Summoned Skeleton Warriors are Permanent and Follow you", want: []want{{name: "RaisedSkeletonPermanentDuration", modType: mod.TypeFlag, value: true, tags: 1}}},
		{line: "Your Blessing Skills are Disabled", want: []want{{name: "DisableSkill", modType: mod.TypeFlag, value: true, tags: 1}}},
		{line: "Your Warcries are disabled", want: []want{{name: "DisableSkill", modType: mod.TypeFlag, value: true, tags: 1}}},
		{line: "Offering Skills have 20% increased Duration", want: []want{{name: "Duration", modType: mod.TypeIncrease, value: 20.0, tags: 1}}},
		{line: "Enemies inflict Elemental Ailments on you instead of nearby Allies", want: []want{{name: "ExtraAura", modType: mod.TypeList}}},
	} {
		e := ParseMod(c.line, false)
		if e == nil || len(e.ModList) == 0 {
			t.Errorf("no mods produced (line is DROPPED WHOLE): %s", c.line)
			continue
		}
		if extra := strings.TrimSpace(modExtra(e)); extra != "" {
			t.Errorf("leftover %q (line is DROPPED WHOLE): %s", extra, c.line)
			continue
		}
		if len(e.ModList) != len(c.want) {
			t.Errorf("%d mods, want %d: %s", len(e.ModList), len(c.want), c.line)
			continue
		}
		for i, w := range c.want {
			m := e.ModList[i]
			if m.Name() != w.name {
				t.Errorf("mod %d name %q, want %q: %s", i, m.Name(), w.name, c.line)
				continue
			}
			if m.Type() != w.modType {
				t.Errorf("mod %d (%s) type %q, want %q: %s", i, w.name, m.Type(), w.modType, c.line)
			}
			switch v := w.value.(type) {
			case float64:
				if got := m.Value().Float(); got != v {
					t.Errorf("mod %d (%s) value %v, want %v: %s", i, w.name, got, v, c.line)
				}
			case bool:
				if got := m.Value().Flag(); got != v {
					t.Errorf("mod %d (%s) flag %v, want %v: %s", i, w.name, got, v, c.line)
				}
			}
			if m.Flags() != w.modFlags {
				t.Errorf("mod %d (%s) ModFlags %d, want %d: %s", i, w.name, m.Flags(), w.modFlags, c.line)
			}
			if m.KeywordFlags() != w.keywordFlags {
				t.Errorf("mod %d (%s) KeywordFlags %d, want %d: %s", i, w.name, m.KeywordFlags(), w.keywordFlags, c.line)
			}
			if got := len(m.Tags()); got != w.tags {
				t.Errorf("mod %d (%s) has %d tags, want %d — a row landed without its tags applies "+
					"UNCONDITIONALLY: %s", i, w.name, got, w.tags, c.line)
			}
		}
	}
}
