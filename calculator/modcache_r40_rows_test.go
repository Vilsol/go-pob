package calculator

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// TestR40ParserRows pins the rows r40 moved to the pin's parse, by NAME|TYPE|VALUE|TAG-COUNT.
func TestR40ParserRows(t *testing.T) {
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		line string
		want []string
	}{
		{"15% increased Duration of Ailments on Enemies", []string{"EnemyAilmentDuration|INC|15|0"}},
		{"10% increased Duration of Elemental Ailments on Enemies", []string{"EnemyElementalAilmentDuration|INC|10|0"}},
		{"20% reduced Duration of Ailments on You", []string{"SelfAilmentDuration|INC|-20|0"}},
		{"50% increased Elemental Ailment Duration on you", []string{"SelfElementalAilmentDuration|INC|50|0"}},
		{"While not on Full Life, Sacrifice 20% of Mana per Second to Recover that much Life", []string{
			"LifeRecovery|BASE|1|2", "ManaDegenPercent|BASE|20|1",
		}},
		{"Base Ignite Duration is 1 second", []string{"IgniteDurationBase|OVERRIDE|1|0"}},
		{"Poisons on you expire 50% slower", []string{"SelfPoisonDebuffExpirationRate|BASE|-50|0"}},
		{"20% reduced Mana Burn rate", []string{"TinctureManaBurnRate|INC|-20|0"}},
		{"All Damage with Maces and Sceptres inflicts Chill", []string{
			"ChaosCanChill|FLAG|1|1", "FireCanChill|FLAG|1|1", "LightningCanChill|FLAG|1|1", "PhysicalCanChill|FLAG|1|1",
		}},
		{"Impale Damage dealt to Enemies Impaled by you ignores Enemy Physical Damage Reduction", []string{"IgnoreEnemyImpalePhysicalDamageReduction|FLAG|1|0"}},
		{"Skills gain Added Chaos Damage equal to 25% of Life Cost, if Life Cost is not higher than the maximum you could spend", []string{
			"ChaosMax|BASE|1|2", "ChaosMin|BASE|1|2",
		}},
		{"15% increased Explicit Life Modifier magnitudes", []string{}},
		{"-2 Prefix Modifiers allowed", []string{}},
		{"Your Chaos Damage can Freeze", []string{"ChaosCanFreeze|FLAG|1|0"}},
		{"Your Elemental Damage can Shock", []string{"ColdCanShock|FLAG|1|0", "FireCanShock|FLAG|1|0"}},
		{"Immune to Elemental Ailments while affected by Glorious Madness", []string{"ElementalAilmentImmune|FLAG|1|1"}},
		{"1.5% of Evasion Rating is Regenerated as Life per second while Focused", []string{"LifeRegen|BASE|1|2"}},
		{"Projectiles Pierce 2 additional Targets if 2 Hunter Items are Equipped", []string{"PierceCount|BASE|2|1"}},
		{"3.0% of Physical Damage Prevented Recently is Regenerated as Energy Shield Per Second if 6 Crusader Items are Equipped", []string{"PhysicalDamageMitigatedEnergyShieldPseudoRecoup|BASE|12|1"}},
		{"50% reduced Enemy Chance to Block Sword Attacks", []string{"reduceEnemyBlock|BASE|50|0"}},
		{"Culling Strike during Effect", []string{"CullPercent|MAX|10|2"}},
		{"Action Speed cannot be modified to below Base Value", []string{"MinimumActionSpeed|MAX|100|1"}},
		{"Cannot be Slowed to below Base Speed", []string{"MinimumActionSpeed|MAX|100|1"}},
		{"Aspect of the Avian also grants Avian's Might and Avian's Flight to nearby Allies", []string{"ExtraSkillMod|LIST|0|1"}},
		{"Your nearby party members maximum Endurance Charges is equal to yours", []string{"PartyMemberMaximumEnduranceChargesEqualToYours|FLAG|1|0"}},
		{"Your Blink and Mirror arrow clones use your Gloves", []string{"BlinkAndMirrorUseGloves|FLAG|1|0"}},
		{"Gain a random Shrine buff every 10 seconds", []string{"Condition:CanHaveRegularShrines|FLAG|1|0"}},
		{"You gain Added Cold Damage instead of Added Damage of other types if Dexterity exceeds both other Attributes", []string{"AllAddedDamageAsCold|FLAG|1|1"}},
		{"You gain Added Lightning Damage instead of Added Damage of other types if Intelligence exceeds both other Attributes", []string{"AllAddedDamageAsLightning|FLAG|1|1"}},
		{"Increases and Reductions to Spell Damage also apply to Attack Damage with Retaliation Skills at 200% of their value", []string{
			"ImprovedSpellDamageAppliesToAttacks|MAX|200|1", "SpellDamageAppliesToAttacks|FLAG|1|1",
		}},
		{"Increases and Reductions to Spell Damage also apply to Attacks while wielding a Wand", []string{"SpellDamageAppliesToAttacks|FLAG|1|1"}},
		{"Inflict Decay on Enemies you Curse with Hex Skills, dealing 700 Chaos Damage per Second for 8 Seconds", []string{"SkillData|LIST|0|1"}},
		{"Gain a Flask Charge when you deal a Critical Strike while affected by Precision", []string{"FlaskChargeOnCritChance|BASE|100|1"}},
		{"While a Unique Enemy is in your Presence, Flasks applied to you have 23% increased Effect", []string{"FlaskEffect|INC|23|2"}},
		{"Brands have 25% increased Area of Effect if 50% of Attached Duration expired", []string{"AreaOfEffect|INC|25|2"}},
		{"Enemies Cursed by you have Malediction if 33% of Curse Duration expired", []string{"EnemyModifier|LIST|0|0"}},
		{"Your Warcries inflict Hallowing Flame", []string{"Condition:CanInflictHallowingFlame|FLAG|1|1"}},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e.Extra != "" {
				t.Fatalf("leftover %q -- an unparsed remainder DISCARDS THE WHOLE LINE", e.Extra)
			}
			got := make([]string, 0, len(e.ModList))
			for _, m := range e.ModList {
				v := m.Value().Float()
				if m.Value().Flag() {
					v = 1
				}
				got = append(got, fmt.Sprintf("%s|%s|%g|%d", m.Name(), m.Type(), v, len(m.Tags())))
			}
			sort.Strings(got)
			want := append([]string(nil), tc.want...)
			sort.Strings(want)
			if strings.Join(got, " ; ") != strings.Join(want, " ; ") {
				t.Fatalf("\n got: %s\nwant: %s", strings.Join(got, " ; "), strings.Join(want, " ; "))
			}
		})
	}
}
