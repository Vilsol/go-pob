package exposition

import (
	"encoding/json"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/pob"
)

func characterWithImbuedSupport(t *testing.T, char string, name string) string {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal([]byte(char), &doc); err != nil {
		t.Fatal(err)
	}
	for _, it := range doc["character"].(map[string]any)["equipment"].([]any) {
		item := it.(map[string]any)
		if item["inventoryId"] == "BodyArmour" {
			item["socketedItems"].([]any)[0].(map[string]any)["builtInSupport"] = "Supported by Level 1 " + name
			out, err := json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			return string(out)
		}
	}
	t.Fatal("REFUSING: the sample character has no body armour")
	return ""
}

func bodyArmourGroup(t *testing.T, build *pob.PathOfBuilding) *pob.Skill {
	t.Helper()
	set := &build.Skills.SkillSets[build.ActiveSkillSetIndex()]
	for i := range set.Skills {
		if set.Skills[i].Slot == "Body Armour" {
			return &set.Skills[i]
		}
	}
	t.Fatal("imported character has no body armour skill group")
	return nil
}

// spec/System/TestImport_spec.lua "imports and reloads an imbued support"
func TestImportImbuedSupportNamesMatchPoB(t *testing.T) {
	initItemData(t)
	_, char := readImportOracle(t)
	for _, c := range []struct{ builtIn, want string }{
		{"Added Fire Damage", "Added Fire Damage"},
		{"Added Fire Damage Support", "Added Fire Damage"},
		{"Barrage Support", "Barrage Support"},
		{"[SupportVolatility|Volatility]", "Volatility"},
		{"Unknown Example", "Unknown Example"},
	} {
		build := builds.EmptyBuild()
		if _, err := ImportItemsAndSkills(build, characterWithImbuedSupport(t, char, c.builtIn), true, true, false); err != nil {
			t.Fatal(err)
		}
		if got := bodyArmourGroup(t, build).ImbuedSupport; got == nil || *got != c.want {
			t.Errorf("%q: imbued support %v, PoB %q", c.builtIn, got, c.want)
		}
	}
}

// spec/System/TestImport_spec.lua "updates imbued support when reimporting an existing skill group"
func TestReimportUpdatesImbuedSupport(t *testing.T) {
	initItemData(t)
	_, char := readImportOracle(t)
	build := builds.EmptyBuild()
	if _, err := ImportItemsAndSkills(build, characterWithImbuedSupport(t, char, "Added Fire Damage"), true, true, false); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ builtIn, want string }{
		{"[UnknownSupport|Unknown Example]", "Unknown Example"},
		{"[SupportVolatility|Volatility]", "Volatility"},
	} {
		if _, err := ImportItemsAndSkills(build, characterWithImbuedSupport(t, char, c.builtIn), false, false, false); err != nil {
			t.Fatal(err)
		}
		if got := bodyArmourGroup(t, build).ImbuedSupport; got == nil || *got != c.want {
			t.Errorf("reimport %q: imbued support %v, PoB %q", c.builtIn, got, c.want)
		}
	}
}
