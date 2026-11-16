package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// gemEditFixture is the shared mid build with socket group 4 in Full DPS, Vaal Double Strike counted three times and
// Impale disabled.
func gemEditFixture(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(fixture("3.xml"))
	if err != nil {
		t.Fatal(err)
	}
	out := string(raw)
	for _, r := range [][2]string{
		{`includeInFullDPS="false" slot="Body Armour"`, `includeInFullDPS="true" slot="Body Armour"`},
		{`<Gem skillId="VaalDoubleStrike" level="20" count="1"`, `<Gem skillId="VaalDoubleStrike" level="20" count="3"`},
		{`gemId="Metadata/Items/Gems/SupportGemImpale" enabled="true" nameSpec="Impale" enableGlobal2="false" enableGlobal1="true" quality="0"/>
				<Gem skillId="SupportRage" level="21" count="1" qualityId="Default" gemId="Metadata/Items/Gems/SupportGemRage" enabled="true" nameSpec="Rage" enableGlobal2="false" enableGlobal1="true" quality="5"`,
			`gemId="Metadata/Items/Gems/SupportGemImpale" enabled="false" nameSpec="Impale" enableGlobal2="false" enableGlobal1="true" quality="0"/>
				<Gem skillId="SupportRage" level="21" count="1" qualityId="Default" gemId="Metadata/Items/Gems/SupportGemRage" enabled="true" nameSpec="Rage" enableGlobal2="false" enableGlobal1="true" quality="5"`},
	} {
		if !strings.Contains(out, r[0]) {
			t.Fatalf("fixture lacks %q", r[0])
		}
		out = strings.Replace(out, r[0], r[1], 1)
	}
	path := filepath.Join(t.TempDir(), "gems.xml")
	if err := os.WriteFile(path, []byte(out), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func groupRow(t *testing.T, s *Server, h string, has string) map[string]any {
	t.Helper()
	res := call(t, s, "get_build", map[string]any{"handle": h, "section": "skills", "detail": true})
	for _, g := range res["groups"].([]any) {
		if row := g.(map[string]any); strings.Contains(fmt.Sprint(row["gems"]), has) {
			return row
		}
	}
	t.Fatalf("no group holds %s", has)
	return nil
}

// TestGemEdits: set_gems keeps a slot's enabled state unless enabled:true, add_gem gives a new active gem the count Full DPS uses for its
// group, and group indexes in a batch name the groups as they were before it.
func TestGemEdits(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "import_build", map[string]any{"source": gemEditFixture(t)})["handle"].(string)

	swap := dryDelta(t, s, "edit_skills", h, "FullDPS", map[string]any{"op": "set_gems", "group": 4,
		"gems": []string{"Rage", "Melee Physical Damage", "Brutality", "Melee Splash", "Double Strike"}})
	res := call(t, s, "edit_skills", map[string]any{"handle": h, "dry_run": true, "watch": []string{"FullDPS"}, "ops": ops(
		map[string]any{"op": "remove_gem", "group": 4, "gem": "Vaal Double Strike"},
		map[string]any{"op": "remove_gem", "group": 4, "gem": "Impale"},
		map[string]any{"op": "add_gem", "group": 4, "gem": "Double Strike"})})
	var add float64
	for _, d := range res["delta"].([]any) {
		if row := d.(map[string]any); row["stat"] == "FullDPS" {
			add = row["after"].(float64) - row["before"].(float64)
		}
	}
	if swap == 0 || add != swap {
		t.Fatalf("remove_gem + add_gem moves FullDPS by %v, set_gems by %v: %v", add, swap, res["applied"])
	}

	impaleEnabled := func() bool {
		for _, g := range groupRow(t, s, h, "Vaal Double Strike")["gems"].([]any) {
			if gem := g.(map[string]any); gem["name"] == "Impale" {
				return gem["enabled"] == true
			}
		}
		t.Fatal("no Impale in the group")
		return false
	}
	res = call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_gems", "group": 4,
		"gems": []string{"Vaal Double Strike", "Impale", "Rage"}})})
	if impaleEnabled() || !strings.Contains(fmt.Sprint(res), "kept Impale disabled") {
		t.Fatalf("set_gems must keep Impale's slot disabled, as Path of Building does, and say so: %v", res)
	}
	res = call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_gems", "group": 4,
		"gems": []string{"Vaal Double Strike", "Impale", "Rage"}, "enabled": true})})
	if !impaleEnabled() {
		t.Fatalf("set_gems enabled:true left Impale disabled: %v", res)
	}

	res = call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "remove_group", "group": 1},
		map[string]any{"op": "set_slot", "group": 3, "slot": "Helmet"},
		map[string]any{"op": "add_group", "gems": []string{"Decoy Totem"}, "slot": "Ring 1"},
		map[string]any{"op": "remove_group", "group": 2},
		map[string]any{"op": "set_slot", "group": 8, "slot": "Ring 2"})})
	if row := groupRow(t, s, h, "Vulnerability"); row["slot"] != "Helmet" {
		t.Fatalf("set_slot group 3 after removing group 1 moved %v, not Vulnerability's group", row)
	}
	if row := groupRow(t, s, h, "Decoy Totem"); row["slot"] != "Ring 2" {
		t.Fatalf("the group added as 8 is %v", row)
	}
	if m := fmt.Sprint(res["group_indexes"]); !strings.Contains(m, "3:1") || !strings.Contains(m, "8:6") {
		t.Fatalf("group_indexes = %v", m)
	}
	e := callErr(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "remove_group", "group": 1}, map[string]any{"op": "enable_group", "group": 1})})
	if e.Code != CodeInvalidValue || !strings.Contains(e.Message, "removed") {
		t.Fatalf("a removed group: %s %s", e.Code, e.Message)
	}
}

// TestGemEditSocketWarningsAndDisable: a fifth gem in four-socket gloves whose gems span two groups is reported in the
// edit result, though one of them (Assassin's Mark) is missing from the game data the engine counts gems against; and
// set_gems enabled:false disables the gems it sets.
func TestGemEditSocketWarningsAndDisable(t *testing.T) {
	s := testServer(t)
	raw, err := os.ReadFile(fixture("3.xml"))
	if err != nil {
		t.Fatal(err)
	}
	from := `<Gem skillId="SupportRage" level="21" count="1" qualityId="Default" gemId="Metadata/Items/Gems/SupportGemRage" enabled="true" nameSpec="Rage" enableGlobal2="false" enableGlobal1="true" quality="0"/>`
	if !strings.Contains(string(raw), from) {
		t.Fatal("fixture lacks the gloves' Rage gem")
	}
	xml := strings.Replace(string(raw), from, `</Skill>
			<Skill enabled="true" slot="Gloves" mainActiveSkill="1">
				<Gem skillId="AssassinsMark" level="20" count="1" gemId="Metadata/Items/Gems/SkillGemAssassinsMark" enabled="true" nameSpec="Assassin's Mark" enableGlobal1="true" quality="0"/>`, 1)
	path := filepath.Join(t.TempDir(), "gloves.xml")
	if err := os.WriteFile(path, []byte(xml), 0o600); err != nil {
		t.Fatal(err)
	}
	h := call(t, s, "import_build", map[string]any{"source": path})["handle"].(string)
	res := call(t, s, "edit_skills", map[string]any{"handle": h, "dry_run": true, "ops": ops(
		map[string]any{"op": "add_gem", "group": 2, "gem": "Melee Physical Damage"})})
	if txt := fmt.Sprint(res["warnings_added"]); !strings.Contains(txt, "too many gems in your Gloves slot") {
		t.Fatalf("add_gem of a fifth gem to 4-socket gloves warns nothing: %v", res["warnings_added"])
	}

	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "set_gems", "group": 2, "gems": []string{"Impale", "Brutality"}, "enabled": false})})
	row := groupRow(t, s, h, "Impale")
	if txt := fmt.Sprint(row); !strings.Contains(txt, "enabled:false") {
		t.Fatalf("set_gems enabled:false left the gems enabled: %v", row)
	}
}

// TestSkillPart: get_build shows a multi-part gem's part and its options, and set_part picks another one, which the
// calculation follows (Storm Burst of Repulsion's own two parts, act_int.lua's parts table, not Storm Burst's three).
func TestSkillPart(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "add_group", "gems": []string{"Storm Burst of Repulsion"}, "slot": "Ring 1"},
		map[string]any{"op": "set_main", "group": 8})})
	row := groupRow(t, s, h, "Storm Burst of Repulsion")
	if txt := fmt.Sprint(row); !strings.Contains(txt, "part:1 Orb Tick") || !strings.Contains(txt, "Max Duration Explode") {
		t.Fatalf("skills row lacks the part and its options: %v", row)
	}
	res := call(t, s, "edit_skills", map[string]any{"handle": h, "dry_run": true, "watch": []string{"TotalDPS"}, "ops": ops(
		map[string]any{"op": "set_part", "group": 8, "part": "max duration explode"})})
	if !strings.Contains(fmt.Sprint(res["applied"]), "Max Duration Explode") || !strings.Contains(fmt.Sprint(res["delta"]), "TotalDPS") {
		t.Fatalf("set_part did not change the part the calculation uses: %v", res)
	}
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_part", "group": 8, "part": 2})})
	if txt := fmt.Sprint(groupRow(t, s, h, "Storm Burst of Repulsion")); !strings.Contains(txt, "part:Max Duration Explode") || strings.Contains(txt, "Max Channelled Orbs") {
		t.Fatalf("set_part by index: %s", txt)
	}
	if e := callErr(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_part", "group": 8, "part": 3})}); !strings.Contains(fmt.Sprint(e.Valid), "Max Duration Explode") {
		t.Fatalf("set_part accepted Storm Burst's third part on Storm Burst of Repulsion: %v", e)
	}
	if e := callErr(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_part", "group": 1, "part": 2})}); !strings.Contains(e.Message, "part") {
		t.Fatalf("set_part on a gem without parts: %s", e.Message)
	}
}

// TestSetGemLevelWithoutGroup: set_gem_level finds the group itself when one gem in the build has the name, and asks
// for the group when several do.
func TestSetGemLevelWithoutGroup(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_gem_level", "gem": "Leap Slam", "level": 5})})
	hh, _ := s.get(h)
	b, _ := hh.parseRev(hh.cur)
	found := false
	for _, g := range activeSkills(b) {
		for _, gem := range g.Gems {
			if gem.NameSpec == "Leap Slam" {
				found = gem.Level == 5
			}
		}
	}
	if !found {
		t.Error("Leap Slam not set to level 5")
	}
	if e := callErr(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_gem_level", "gem": "Impale", "level": 5})}); e.Code != CodeAmbiguous || len(e.Candidates) != 2 {
		t.Errorf("Impale in two groups: %s %v", e.Code, e.Candidates)
	}
	if e := callErr(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_gem_level", "gem": "Fireball", "level": 5})}); e.Code != CodeNotFound {
		t.Errorf("no Fireball: %s", e.Code)
	}
}

// TestSetGemsKeepsSkillPart: set_gems changing a slot's gem keeps the slot's skill part, as PoB's skills tab changes the
// gem in the same gem instance (Storm Burst on its second part becomes Storm Burst of Repulsion on its second part).
func TestSetGemsKeepsSkillPart(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "add_group", "gems": []string{"Storm Burst"}, "slot": "Ring 1"},
		map[string]any{"op": "set_main", "group": 8},
		map[string]any{"op": "set_part", "group": 8, "part": 2})})
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_gems", "group": 8, "gems": []string{"Storm Burst of Repulsion"}})})
	if txt := fmt.Sprint(groupRow(t, s, h, "Storm Burst of Repulsion")); !strings.Contains(txt, "part:Max Duration Explode") {
		t.Fatalf("set_gems reset the skill part: %s", txt)
	}
}

// TestNewGemsAtCharacterLevel: a gem added without a level takes the highest level a level 30 character meets the
// requirement of, not the build's default 20.
func TestNewGemsAtCharacterLevel(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Templar", "level": 30})["handle"].(string)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "add_group", "gems": []string{"Freezing Pulse"}})})
	lv := (*groupsOf(mustBuild(t, s, h)))[0].Gems[0].Level
	if gemLevelReq("FreezingPulse", lv) > 30 || gemLevelReq("FreezingPulse", lv+1) <= 30 {
		t.Fatalf("Freezing Pulse added at level %d (needs %d; %d needs %d)", lv, gemLevelReq("FreezingPulse", lv), lv+1, gemLevelReq("FreezingPulse", lv+1))
	}
}
