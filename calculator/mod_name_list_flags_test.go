package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestModNameListRowsCarryTheirFlags pins the `flags` / `keywordFlags` that eighteen modNameList rows drop.
func TestModNameListRowsCarryTheirFlags(t *testing.T) {
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		line   string
		pinRow string
		want   []struct {
			name  string
			flags mod.MFlag
			kw    mod.KeywordFlag
		}
	}{
		{"125% increased Damage with Poison", "737", []struct {
			name  string
			flags mod.MFlag
			kw    mod.KeywordFlag
		}{{"Damage", 0, mod.KeywordFlagPoison}}},
		{"15% increased Damage with Bleeding", "736", []struct {
			name  string
			flags mod.MFlag
			kw    mod.KeywordFlag
		}{{"Damage", 0, mod.KeywordFlagBleed}}},
		{"15% increased Damage with Bleed", "735", []struct {
			name  string
			flags mod.MFlag
			kw    mod.KeywordFlag
		}{{"Damage", 0, mod.KeywordFlagBleed}}},
		{"30% increased Damage with Ignites inflicted", "734", []struct {
			name  string
			flags mod.MFlag
			kw    mod.KeywordFlag
		}{{"Damage", 0, mod.KeywordFlagIgnite}}},

		{"50% increased Effect of Auras from your Vaal Skills", "464", []struct {
			name  string
			flags mod.MFlag
			kw    mod.KeywordFlag
		}{{"AuraEffect", 0, mod.KeywordFlagVaal}}},

		{"Bow Attacks fire 2 additional Arrows", "590", []struct {
			name  string
			flags mod.MFlag
			kw    mod.KeywordFlag
		}{{"ProjectileCount", 0, mod.KeywordFlagArrow}}},

		{"Bow Attacks fire an additional Arrow", "not the arrow row — ModFlag.Bow, kw 0", []struct {
			name  string
			flags mod.MFlag
			kw    mod.KeywordFlag
		}{{"ProjectileCount", mod.MFlagBow, 0}}},

		{"Gain 5 Life for each Enemy hit", "537", []struct {
			name  string
			flags mod.MFlag
			kw    mod.KeywordFlag
		}{{"LifeOnHit", mod.MFlagHit, 0}}},
		{"Gain 5 Life per Enemy Hit", "535", []struct {
			name  string
			flags mod.MFlag
			kw    mod.KeywordFlag
		}{{"LifeOnHit", mod.MFlagHit, 0}}},
		{"+5 Life gained for each Enemy hit", "536", []struct {
			name  string
			flags mod.MFlag
			kw    mod.KeywordFlag
		}{{"LifeOnHit", mod.MFlagHit, 0}}},
		{"Gain 4 Life and Mana for each Enemy hit", "551", []struct {
			name  string
			flags mod.MFlag
			kw    mod.KeywordFlag
		}{{"LifeOnHit", mod.MFlagHit, 0}, {"ManaOnHit", mod.MFlagHit, 0}}},
		{"Gain 3 Mana for each Enemy hit", "543", []struct {
			name  string
			flags mod.MFlag
			kw    mod.KeywordFlag
		}{{"ManaOnHit", mod.MFlagHit, 0}}},
		{"+3 Mana per Enemy Hit", "541", []struct {
			name  string
			flags mod.MFlag
			kw    mod.KeywordFlag
		}{{"ManaOnHit", mod.MFlagHit, 0}}},
		{"+7 Mana gained for each Enemy hit", "542", []struct {
			name  string
			flags mod.MFlag
			kw    mod.KeywordFlag
		}{{"ManaOnHit", mod.MFlagHit, 0}}},
		{"Gain 12 Energy Shield for each Enemy hit", "549", []struct {
			name  string
			flags mod.MFlag
			kw    mod.KeywordFlag
		}{{"EnergyShieldOnHit", mod.MFlagHit, 0}}},
		{"Gain 12 Energy Shield per Enemy Hit", "547", []struct {
			name  string
			flags mod.MFlag
			kw    mod.KeywordFlag
		}{{"EnergyShieldOnHit", mod.MFlagHit, 0}}},
		{"+9 Energy Shield gained for each Enemy hit", "548", []struct {
			name  string
			flags mod.MFlag
			kw    mod.KeywordFlag
		}{{"EnergyShieldOnHit", mod.MFlagHit, 0}}},

		{"+25% chance to Impale on Attack Hit", "579", []struct {
			name  string
			flags mod.MFlag
			kw    mod.KeywordFlag
		}{{"ImpaleChance", mod.MFlagAttack, 0}}},
		{"+25% chance to Impale on Spell Hit", "580 (control — already correct)", []struct {
			name  string
			flags mod.MFlag
			kw    mod.KeywordFlag
		}{{"ImpaleChance", mod.MFlagSpell, 0}}},
	} {
		t.Run(tc.line, func(t *testing.T) {
			entry := ParseMod(tc.line, false)
			if entry.Extra != "" {
				t.Fatalf("ModParser.lua:%s — %q left %q unparsed; an unparsed remainder discards "+
					"the whole line", tc.pinRow, tc.line, entry.Extra)
			}
			if len(entry.ModList) != len(tc.want) {
				t.Fatalf("ModParser.lua:%s — %q produced %d mods, want %d",
					tc.pinRow, tc.line, len(entry.ModList), len(tc.want))
			}
			for i, want := range tc.want {
				got := entry.ModList[i]
				if got.Name() != want.name {
					t.Errorf("mod %d: name = %s, want %s", i, got.Name(), want.name)
				}
				if got.Flags() != want.flags {
					t.Errorf("ModParser.lua:%s — %q mod %d (%s): flags = %#x, want %#x",
						tc.pinRow, tc.line, i, want.name, got.Flags(), want.flags)
				}
				if got.KeywordFlags() != want.kw {
					t.Errorf("ModParser.lua:%s — %q mod %d (%s): keywordFlags = %#x, want %#x",
						tc.pinRow, tc.line, i, want.name, got.KeywordFlags(), want.kw)
				}
			}
		})
	}
}
