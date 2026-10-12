package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestBannerAuraPrefixNarrowsToBanners pins ModParser.lua:1291 and :6871-6874.
func TestBannerAuraPrefixNarrowsToBanners(t *testing.T) {
	bannerTags := func(t *testing.T, line string) (int, int) {
		t.Helper()
		entry := ParseMod(line, false)
		if entry.Extra != "" {
			t.Fatalf("unparsed remainder %q", entry.Extra)
		}
		if len(entry.ModList) != 1 {
			t.Fatalf("got %d mods, want 1", len(entry.ModList))
		}
		m := entry.ModList[0]
		if m.Name() != "ExtraAuraEffect" {
			t.Fatalf("got %s, want ExtraAuraEffect", m.Name())
		}
		banners := 0
		for _, tg := range m.Tags() {
			if st, ok := tg.(*mod.SkillTypeTag); ok {
				for _, v := range st.SkillTypeList {
					if v == "Banner" {
						banners++
					}
				}
			}
		}
		return banners, len(m.Tags())
	}

	if got, total := bannerTags(t, "Banners also grant 10% increased Attack Damage"); got != 1 {
		t.Errorf("banner row carried %d Banner SkillType tags of %d, want 1", got, total)
	}
	if got, total := bannerTags(t, "Auras grant 10% increased Attack Damage"); got != 0 {
		t.Errorf("plain aura row carried %d Banner SkillType tags of %d, want 0", got, total)
	}
}

// TestAffectedByAuraUsesAConditionTag pins ModParser.lua:1297 and :1328.
func TestAffectedByAuraUsesAConditionTag(t *testing.T) {
	entry := ParseMod("You and Allies affected by Auras from your Skills have 10% increased Attack Damage", false)
	if entry.Extra != "" {
		t.Fatalf("unparsed remainder %q", entry.Extra)
	}
	if len(entry.ModList) != 1 {
		t.Fatalf("got %d mods, want 1", len(entry.ModList))
	}
	m := entry.ModList[0]
	if m.Name() != "Damage" || m.Type() != mod.TypeIncrease {
		t.Fatalf("got %s %s, want Damage INC -- an AffectedByAuraMod wrapper here is the defect", m.Name(), m.Type())
	}
	var saw bool
	for _, tg := range m.Tags() {
		if c, ok := tg.(*mod.ConditionTag); ok {
			for _, v := range c.VarList {
				if v == "AffectedByAura" {
					saw = true
				}
			}
		}
	}
	if !saw {
		t.Errorf("no Condition:AffectedByAura tag; tags are %v", m.Tags())
	}
}
