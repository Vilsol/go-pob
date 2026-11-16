package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// auditFixture is the shared mid build with Berserk at quality 30, Brutality at 22 and Rage swapped for the legacy
// Awakened Added Fire Damage, plus a custom resistance line.
func auditFixture(t *testing.T, extra ...[2]string) string {
	t.Helper()
	raw, err := os.ReadFile(fixture("3.xml"))
	if err != nil {
		t.Fatal(err)
	}
	out := string(raw)
	for _, r := range append([][2]string{
		{`nameSpec="Berserk" enableGlobal2="false" enableGlobal1="true" quality="7"`, `nameSpec="Berserk" enableGlobal2="false" enableGlobal1="true" quality="30"`},
		{`nameSpec="Brutality" enableGlobal2="false" enableGlobal1="true" quality="11"`, `nameSpec="Brutality" enableGlobal2="false" enableGlobal1="true" quality="22"`},
		{`<Gem skillId="SupportRage" level="21" count="1" qualityId="Default" gemId="Metadata/Items/Gems/SupportGemRage" enabled="true" nameSpec="Rage"`,
			`<Gem skillId="SupportAwakenedAddedFireDamage" level="5" count="1" qualityId="Default" gemId="Metadata/Items/Gems/SupportGemAddedFireDamagePlus" enabled="true" nameSpec="Awakened Added Fire Damage"`},
		{"\t</Config>", `<CustomModifierBlock title="Default" enabled="true">+30% to Fire Resistance</CustomModifierBlock>` + "\n\t</Config>"},
	}, extra...) {
		if !strings.Contains(out, r[0]) {
			t.Fatalf("fixture lacks %q", r[0])
		}
		out = strings.Replace(out, r[0], r[1], 1)
	}
	path := filepath.Join(t.TempDir(), "audit.xml")
	if err := os.WriteFile(path, []byte(out), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestImportAuditFlagsGemQualityLegacyGemsCustomModsAndConfigLimits(t *testing.T) {
	s := testServer(t)
	res := call(t, s, "import_build", map[string]any{"source": auditFixture(t)})
	h := res["handle"].(string)
	audit := fmt.Sprint(res["audit"])
	for _, want := range []string{"Berserk", "quality:30", "above the game's 23% maximum", "Brutality", "quality:22", "needs a corrupted gem",
		"Awakened Added Fire Damage", "Added Fire Damage", "Fire Res +30%"} {
		if !strings.Contains(audit, want) {
			t.Fatalf("audit lacks %q: %s", want, audit)
		}
	}
	if strings.Contains(audit, "Vitality") {
		t.Fatalf("audit flags a quality-0 gem: %s", audit)
	}
	warnings := fmt.Sprint(res["summary"].(map[string]any)["warnings"])
	if !strings.Contains(warnings, "Berserk in socket group 1 has 30% quality") || strings.Contains(warnings, "Brutality in socket group") {
		t.Fatalf("warnings: %s", warnings)
	}

	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "add_group", "gems": []string{"Shockwave Totem"}, "slot": "Ring 1"},
		map[string]any{"op": "add_group", "gems": []string{"Decoy Totem"}, "slot": "Ring 2"}, map[string]any{"op": "set_main", "group": 8})})
	call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_config", "var": "TotemsSummoned", "value": 9})})
	got := call(t, s, "get_build", map[string]any{"handle": h, "section": "warnings"})
	audit = fmt.Sprint(got["audit"])
	for _, want := range []string{"Config sets 9 totems summoned", "Decoy Totem also summon totems", "Berserk"} {
		if !strings.Contains(audit, want) {
			t.Fatalf("warnings-section audit lacks %q: %s", want, audit)
		}
	}

	slot := map[string]any{"kind": "gem", "group": 1, "gem": "Berserk"}
	opt := call(t, s, "optimize", map[string]any{"handle": h, "objective": map[string]any{"stat": "TotalEHP"}, "top": 5, "slots": []any{slot}})
	if txt := fmt.Sprint(opt); !strings.Contains(txt, "quality 30 (above the game's 23% maximum)") {
		t.Fatalf("optimize does not mark the imported quality 30 impossible: %s", txt)
	}
}

// TestImportAuditWidening covers a name-only legacy gem, custom maximum resistance and damage reduction, a stored
// Rage above the maximum PoB caps it at, gems in another skill set and a Delve drop-only mod.
func TestImportAuditWidening(t *testing.T) {
	s := testServer(t)
	path := auditFixture(t,
		[2]string{`<Gem skillId="SupportMeleeSplash" level="21" count="1" qualityId="Default" gemId="Metadata/Items/Gems/SupportGemMeleeSplash" enabled="true" nameSpec="Melee Splash"`,
			`<Gem level="5" count="1" enabled="true" nameSpec="Awakened Lightning Penetration"`},
		[2]string{"+30% to Fire Resistance</CustomModifierBlock>", "+30% to Fire Resistance\n+2% to maximum Fire Resistance\n5% additional Physical Damage Reduction</CustomModifierBlock>"},
		[2]string{"\t\t</SkillSet>\n\t</Skills>", "\t\t</SkillSet>\n\t\t<SkillSet id=\"2\" title=\"Min-Max\">\n\t\t\t<Skill enabled=\"true\" slot=\"Helmet\" mainActiveSkill=\"1\">\n" +
			`<Gem skillId="Vitality" level="20" count="1" gemId="Metadata/Items/Gems/SkillGemVitality" enabled="true" nameSpec="Vitality" enableGlobal1="true" quality="32"/>` +
			"\n\t\t\t</Skill>\n\t\t</SkillSet>\n\t</Skills>"},
		[2]string{"+46 to Intelligence\n", "+46 to Intelligence\n0.4% of Chaos Damage Leeched as Life\n"},
	)
	res := call(t, s, "import_build", map[string]any{"source": path})
	audit := fmt.Sprint(res["audit"])
	for _, want := range []string{"Awakened Lightning Penetration", "Lightning Penetration",
		"Fire Res Max +2%", "Phys Damage Reduction +5%",
		"multiplierRage is 120", "set:Min-Max", "quality:32",
		"0.4% of Chaos Damage Leeched as Life", "Ring 1"} {
		if !strings.Contains(audit, want) {
			t.Fatalf("audit lacks %q: %s", want, audit)
		}
	}
	if warnings := fmt.Sprint(res["summary"].(map[string]any)["warnings"]); strings.Contains(warnings, "Vitality in socket group") {
		t.Fatalf("summary warnings name a gem outside the active skill set: %s", warnings)
	}
}

// TestImportAuditGemLevels: a gem one level above its natural maximum needs a corrupted gem, more is impossible, and
// a gem whose level and quality both need the corruption is called out, since a gem is corrupted once.
func TestImportAuditGemLevels(t *testing.T) {
	s := testServer(t)
	path := auditFixture(t,
		[2]string{`<Gem skillId="SupportBrutality" level="20"`, `<Gem skillId="SupportBrutality" level="21"`},
		[2]string{`level="20" count="1" qualityId="Default" gemId="Metadata/Items/Gems/SkillGemVitality"`, `level="23" count="1" qualityId="Default" gemId="Metadata/Items/Gems/SkillGemVitality"`},
	)
	res := call(t, s, "import_build", map[string]any{"source": path})
	audit := res["audit"].(map[string]any)
	levels := fmt.Sprint(audit["gem_levels"])
	for _, want := range []string{"level 21 needs a corrupted gem: Berserk (1)", "level 23 above the game's maximum level 21 (corrupted): Vitality (1)"} {
		if !strings.Contains(levels, want) {
			t.Fatalf("gem_levels lacks %q: %s", want, levels)
		}
	}
	if txt := fmt.Sprint(audit["gem_quality"]); !strings.Contains(txt, "level 21 and quality 22% each need a corruption") {
		t.Fatalf("Brutality 21/22 is not called out: %s", txt)
	}
	if warnings := fmt.Sprint(res["summary"].(map[string]any)["warnings"]); !strings.Contains(warnings, "Vitality in socket group 1 is level 23") {
		t.Fatalf("summary warnings: %s", warnings)
	}
}
