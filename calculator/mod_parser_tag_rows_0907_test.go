package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestModTagListRows0907 pins the modTagList rows found by re-deriving the two-sided name-set diff against ModParser.lua:1319-2008.
func TestModTagListRows0907(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("10% increased Damage per Power Charge", false); e == nil ||
		len(e.ModList) == 0 || strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: an already-ported modTagList row no longer parses")
	}

	for _, tc := range []struct {
		name    string
		line    string
		wantTag string
	}{
		{"critKill", "40% increased Rarity of Items Dropped by Enemies killed with a Critical Strike",
			"Condition/vars=CritRecently"},
		{"socketedGem", "+1 to Level of Socketed Skill Gems per Socketed Gem",
			"Multiplier/vars=SocketedGemsIn{SlotName}"},
		{"nebulis", "13% increased Elemental Damage per 1% Missing Fire, Cold, or Lightning Resistance, up to a maximum of 450%",
			"PerStat/globalLimit=450/globalLimitKey=ReplicaNebulisCold/stats=MissingFireResist,MissingColdResist,MissingLightningResist"},
		{"noFlasks", "You have Brutal Shrine Buff while affected by no Flasks",
			"Condition/neg=true/vars=UsingFlask"},
		{"twoFort", "Nearby Enemies take 10% increased Physical Damage per two Fortification on you",
			"PerStat/actor=player/div=2/stats=FortificationStacks"},
		{"meleeKill", "5% chance to Gain Unholy Might for 4 seconds on Melee Kill",
			"Condition/vars=KilledRecently"},
		{"youStationary", "Nearby Enemy Monsters' Fire Resistance against Damage over Time is -20% while you are Stationary",
			"ActorCondition/actor=player/vars=Stationary"},
		{"manaBurn", "5% increased Rarity of Items found per Mana Burn, up to a maximum of 100%",
			"Multiplier/limit=100/limitTotal=true/vars=ManaBurnStacks"},
		{"shockedNonShocked", "10% increased Burning Damage for each time you have Shocked a Non-Shocked Enemy Recently, up to a maximum of 120%",
			"Multiplier/limit=120/limitTotal=true/vars=ShockedNonShockedEnemyRecently"},
		{"playerES", "Regenerate 1% of Life per second per 500 Player Maximum Energy Shield",
			"PerStat/actor=player/div=500/stats=EnergyShield"},
		{"onCursed", "30% increased Damage with Ignites inflicted on Cursed Enemies",
			"ActorCondition/actor=enemy/vars=Cursed"},
		{"excommunicated", "Melee Attacks have +10% to Critical Strike Chance against Excommunicated Enemies",
			"ActorCondition/actor=enemy/vars=Excommunicated"},
		{"higherLife", "50% increased Attack Damage against Enemies with a higher percentage of their Life remaining than you",
			"ActorCondition/actor=enemy/vars=HigherLifePercentThanPlayer"},
		{"curseExpired", "Enemies Cursed by you take 20% increased Damage if 75% of Curse Duration expired",
			"MultiplierThreshold/actor=enemy/threshold=75/vars=CurseExpired"},
		{"notCloseRange", "50% more Damage with Arrow Hits not at Close Range",
			"Condition/neg=true/vars=AtCloseRange"},

		{"nb_manaBurn", "5% increased Rarity of Items found per Mana Burn",
			"Multiplier/vars=ManaBurnStacks"},
		{"nb_socketedRedGem", "Attack Skills gain 5% of Physical Damage as Extra Fire Damage per Socketed Red Gem",
			"SocketedIn/keyword=strength/slotName={SlotName}/sockets={1,2,3,4,5,6}"},
		{"nb_onKill", "10% increased Movement Speed on Kill",
			"Condition/vars=KilledRecently"},
		{"nb_withFlasks", "10% increased Damage while affected by Flasks",
			"Condition/vars=UsingFlask"},
		{"nb_fortification", "3% increased Armour per Fortification",
			"PerStat/stats=FortificationStacks"},
		{"nb_stationary", "10% increased Damage while Stationary",
			"Condition/vars=Stationary"},
		{"nb_closeRange", "10% more Damage with Hits at Close Range",
			"Condition/vars=AtCloseRange"},
		{"nb_againstCursed", "10% increased Damage against Cursed Enemies",
			"ActorCondition/actor=enemy/vars=Cursed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil || len(e.ModList) == 0 {
				t.Fatalf("no mods at all (the line is DROPPED WHOLE): %s", tc.line)
			}
			if extra := strings.TrimSpace(modExtra(e)); extra != "" {
				t.Fatalf("leftover %q — an unparsed remainder discards the WHOLE line: %s", extra, tc.line)
			}
			got := collectCanonTags(e.ModList)
			if !contains(got, tc.wantTag) {
				t.Fatalf("tags %v, want %q: %s", got, tc.wantTag, tc.line)
			}
		})
	}
}

func collectCanonTags(mods []mod.Mod) []string {
	var out []string
	for _, m := range mods {
		if m == nil {
			continue
		}
		for _, tag := range m.Tags() {
			out = append(out, canonOurTag(tag).String())
		}
		if v := m.Value(); v != nil {
			switch inner := v.ValueList.(type) {
			case mod.EnemyModifier:
				out = append(out, collectCanonTags([]mod.Mod{inner.Mod})...)
			case mod.MinionModifier:
				out = append(out, collectCanonTags([]mod.Mod{inner.Mod})...)
			case mod.ExtraSkillMod:
				out = append(out, collectCanonTags([]mod.Mod{inner.Mod})...)
			}
		}
	}
	return out
}

// TestMeleeKillRowCarriesTheMeleeFlag is split out because ModParser.lua:1800 is the only row in the batch whose payload is a FLAG rather than a tag.
func TestMeleeKillRowCarriesTheMeleeFlag(t *testing.T) {
	requireGameData(t, "3.29")
	e := ParseMod("5% chance to Gain Unholy Might for 4 seconds on Melee Kill", false)
	if e == nil || len(e.ModList) == 0 {
		t.Fatal("line dropped whole")
	}
	for _, m := range e.ModList {
		if m.Flags()&mod.MFlagWeaponMelee == 0 {
			t.Fatalf("mod %q lost ModFlag.WeaponMelee (ModParser.lua:1800 sets it); flags=%d",
				m.Name(), m.Flags())
		}
	}
}
