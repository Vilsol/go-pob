package exposition

import (
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/pob"
)

type importOracleGem struct {
	NameSpec string `json:"nameSpec"`
	Level    int    `json:"level"`
	Quality  int    `json:"quality"`
	Enabled  bool   `json:"enabled"`
	SkillID  string `json:"skillId"`
}

type importOracleState struct {
	Items []struct {
		ID  int    `json:"id"`
		Raw string `json:"raw"`
	} `json:"items"`
	Slots  map[string]int `json:"slots"`
	Groups []struct {
		Label         string            `json:"label"`
		Enabled       bool              `json:"enabled"`
		Slot          string            `json:"slot"`
		ImbuedSupport string            `json:"imbuedSupport"`
		Gems          []importOracleGem `json:"gems"`
	} `json:"groups"`
	MainSocketGroup int `json:"mainSocketGroup"`
	Level           int `json:"level"`
}

type importOracle struct {
	First          importOracleState `json:"first"`
	Second         importOracleState `json:"second"`
	Third          importOracleState `json:"third"`
	Onto           importOracleState `json:"onto"`
	GemForBaseName map[string]string `json:"gemForBaseName"`
	VaalGemFor     map[string]string `json:"vaalGemFor"`
}

func readImportOracle(t *testing.T) (importOracle, string) {
	t.Helper()
	dir := filepath.Join("..", "..", "testdata", "import")
	var oracle importOracle
	b, err := os.ReadFile(filepath.Join(dir, "import_items_and_skills.json"))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	if err := json.Unmarshal(b, &oracle); err != nil {
		t.Fatal(err)
	}
	char, err := os.ReadFile(filepath.Join(dir, "sample_character.json"))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	if len(oracle.First.Items) == 0 || len(oracle.First.Groups) == 0 {
		t.Fatal("REFUSING: the oracle imported nothing")
	}
	return oracle, string(char)
}

func gradeImport(t *testing.T, label string, build *pob.PathOfBuilding, want importOracleState, kept map[int]bool) {
	t.Helper()
	if len(build.Items.Items) != len(want.Items) {
		t.Errorf("%s: %d items, PoB %d", label, len(build.Items.Items), len(want.Items))
	}
	for k, item := range build.Items.Items {
		if k >= len(want.Items) {
			break
		}
		if item.ID != want.Items[k].ID || (!kept[item.ID] && html.UnescapeString(item.Text) != want.Items[k].Raw) {
			t.Errorf("%s: item %d (id %d):\n%s\nPoB (id %d):\n%s", label, k, item.ID, html.UnescapeString(item.Text), want.Items[k].ID, want.Items[k].Raw)
		}
	}
	got := map[string]int{}
	for _, set := range build.Items.ItemSets {
		if id, _ := strconv.Atoi(set.ID); id == build.Items.ActiveItemSet {
			for _, slot := range set.Slots {
				if slot.ItemID != 0 {
					got[slot.Name] = slot.ItemID
				}
			}
		}
	}
	for name, id := range want.Slots {
		if got[name] != id {
			t.Errorf("%s: slot %s holds %d, PoB %d", label, name, got[name], id)
		}
	}
	if len(got) != len(want.Slots) {
		t.Errorf("%s: %d slots filled, PoB %d", label, len(got), len(want.Slots))
	}
	groups := build.Skills.SkillSets[build.ActiveSkillSetIndex()].Skills
	if len(groups) != len(want.Groups) {
		t.Fatalf("%s: %d socket groups, PoB %d", label, len(groups), len(want.Groups))
	}
	for g, group := range groups {
		w := want.Groups[g]
		imbued := ""
		if group.ImbuedSupport != nil {
			imbued = *group.ImbuedSupport
		}
		if group.Label != w.Label || group.Enabled != w.Enabled || group.Slot != w.Slot || imbued != w.ImbuedSupport || len(group.Gems) != len(w.Gems) {
			t.Errorf("%s: group %d %+v, PoB %+v", label, g+1, group, w)
			continue
		}
		for k, gem := range group.Gems {
			if got := (importOracleGem{gem.NameSpec, gem.Level, gem.Quality, gem.Enabled, gem.SkillID}); got != w.Gems[k] {
				t.Errorf("%s: group %d gem %d %+v, PoB %+v", label, g+1, k+1, got, w.Gems[k])
			}
		}
	}
	if build.Build.MainSocketGroup != want.MainSocketGroup || build.Build.Level != want.Level {
		t.Errorf("%s: main group %d level %d, PoB %d / %d", label, build.Build.MainSocketGroup, build.Build.Level, want.MainSocketGroup, want.Level)
	}
}

// TestImportItemsAndSkillsAgainstPoB: ImportTab.lua:1380-1490 on PoB's sample character.
func TestImportItemsAndSkillsAgainstPoB(t *testing.T) {
	initItemData(t)
	oracle, char := readImportOracle(t)
	build := builds.EmptyBuild()
	build.Build.CharacterLevelAutoMode = false
	if msg, err := ImportItemsAndSkills(build, char, true, true, false); err != nil || msg != "Items and skills successfully imported." {
		t.Fatalf("first import: %q %v", msg, err)
	}
	gradeImport(t, "first", build, oracle.First, nil)
	if _, err := ImportItemsAndSkills(build, char, true, true, false); err != nil {
		t.Fatal(err)
	}
	gradeImport(t, "second", build, oracle.Second, nil)
	if _, err := ImportItemsAndSkills(build, char, false, false, false); err != nil {
		t.Fatal(err)
	}
	gradeImport(t, "third", build, oracle.Third, nil)

	onto := corpusBuild(t, "0280-elementalist-fireball.xml")
	onto.Build.CharacterLevelAutoMode = false
	kept := map[int]bool{}
	for _, item := range onto.Items.Items {
		kept[item.ID] = true
	}
	if _, err := ImportItemsAndSkills(onto, char, true, true, false); err != nil {
		t.Fatal(err)
	}
	gradeImport(t, "onto 0280", onto, oracle.Onto, kept)
}

// TestImportGemTablesAgainstPoB: Data.lua:1057-1101: every data.gemForBaseName key resolves to PoB's gem, and every Vaal gem lookup to PoB's Vaal gem.
func TestImportGemTablesAgainstPoB(t *testing.T) {
	initItemData(t)
	oracle, _ := readImportOracle(t)
	if len(oracle.GemForBaseName) < 500 || len(oracle.VaalGemFor) < 50 {
		t.Fatalf("REFUSING: %d base names, %d Vaal gems", len(oracle.GemForBaseName), len(oracle.VaalGemFor))
	}
	tables := importGemLookup()
	for name, skillID := range oracle.GemForBaseName {
		if gem, ok := tables.forBaseName[name]; !ok || gem.skillID != skillID {
			t.Errorf("gemForBaseName[%q] = %+v, PoB %s", name, gem, skillID)
		}
	}
	for name := range tables.forBaseName {
		if _, ok := oracle.GemForBaseName[name]; !ok {
			t.Errorf("gemForBaseName[%q] is not PoB's", name)
		}
	}
	for skillID, name := range oracle.VaalGemFor {
		if gem := tables.forVaalGem[skillID]; gem.name != name {
			t.Errorf("gemGrantedEffectIdForVaalGemId[%q] = %q, PoB %q", skillID, gem.name, name)
		}
	}
	if len(tables.forVaalGem) != len(oracle.VaalGemFor) {
		t.Errorf("%d Vaal gem lookups, PoB %d", len(tables.forVaalGem), len(oracle.VaalGemFor))
	}
}

// TestImportItemsAndSkillsRefusals: ImportTab.lua:609-628: what the import code box says about a JSON that is not a character.
func TestImportItemsAndSkillsRefusals(t *testing.T) {
	build := builds.EmptyBuild()
	for text, want := range map[string]string{
		`{`:                                  "Invalid JSON format (decode error)",
		`{"x":1}`:                            "Invalid JSON format (character missing)",
		`{"character":{"equipment":[]}}`:     "Invalid JSON format (equipment or passives missing)",
		`{"character":{"passives":{"x":1}}}`: "Invalid JSON format (equipment or passives missing)",
	} {
		if _, err := ImportItemsAndSkills(build, text, true, true, false); err == nil || err.Error() != want {
			t.Errorf("%s: %v, want %q", text, err, want)
		}
	}
}

// TestImportDuplicatedItemIsMirrored: ImportTab.lua:1641 (#10237): the API marks a mirrored copy "duplicated"; it imports as Mirrored.
func TestImportDuplicatedItemIsMirrored(t *testing.T) {
	_, char := readImportOracle(t)
	var doc map[string]any
	if err := json.Unmarshal([]byte(char), &doc); err != nil {
		t.Fatal(err)
	}
	equipment := doc["character"].(map[string]any)["equipment"].([]any)
	for _, duplicated := range []bool{false, true} {
		item := equipment[0].(map[string]any)
		item["duplicated"] = duplicated
		delete(item, "mirrored")
		text, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		build := builds.EmptyBuild()
		if _, err := ImportItemsAndSkills(build, string(text), true, true, false); err != nil {
			t.Fatal(err)
		}
		mirrored := 0
		for _, it := range build.Items.Items {
			for _, line := range strings.Split(html.UnescapeString(it.Text), "\n") {
				if strings.TrimSpace(line) == "Mirrored" {
					mirrored++
				}
			}
		}
		if want := map[bool]int{false: 0, true: 1}[duplicated]; mirrored != want {
			t.Errorf("duplicated %v: %d items Mirrored, want %d", duplicated, mirrored, want)
		}
	}
}
