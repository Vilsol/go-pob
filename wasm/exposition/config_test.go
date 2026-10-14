package exposition

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/storage"
)

func initGameData(t *testing.T) {
	t.Helper()
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
}

type customModLineExpected struct {
	Line   string `json:"line"`
	Parsed bool   `json:"parsed"`
	Mods   []struct {
		Name  string   `json:"name"`
		Type  string   `json:"type"`
		Value string   `json:"value"`
		Tags  []string `json:"tags"`
	} `json:"mods"`
}

var knownModLineDivergences = map[string]bool{}

// TestParseModLinesAgainstPoB: Graded against oracle/custom_mod_lines.lua: PoB's own verdict on the same lines.
func TestParseModLinesAgainstPoB(t *testing.T) {
	initGameData(t)

	blob, err := os.ReadFile(filepath.Join("..", "..", "testdata", "oracle", "custom_mod_lines_expected.json"))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	var want []customModLineExpected
	if err := json.Unmarshal(blob, &want); err != nil {
		t.Fatal(err)
	}
	var parsed, failed int
	for _, w := range want {
		if w.Parsed {
			parsed++
		} else {
			failed++
		}
	}
	if parsed == 0 || failed == 0 {
		t.Fatalf("REFUSING: the fixture needs both verdicts (parsed %d, not parsed %d)", parsed, failed)
	}

	lines := make([]string, len(want))
	for i, w := range want {
		lines[i] = w.Line
	}
	got := ParseModLines(lines)
	if len(got) != len(want) {
		t.Fatalf("%d verdicts for %d lines", len(got), len(want))
	}

	for i, w := range want {
		g := got[i]
		if knownModLineDivergences[w.Line] {
			if (g.Status == ModLineParsed) == w.Parsed {
				t.Errorf("%q now agrees with PoB: retire it from knownModLineDivergences", w.Line)
			}
			continue
		}
		if (g.Status == ModLineParsed) != w.Parsed {
			t.Errorf("%q: status %s, PoB parsed=%v", w.Line, g.Status, w.Parsed)
			continue
		}
		if !w.Parsed {
			continue
		}
		if len(g.Mods) != len(w.Mods) {
			t.Errorf("%q: %d mods, PoB %d", w.Line, len(g.Mods), len(w.Mods))
			continue
		}
		for j, wm := range w.Mods {
			gm := g.Mods[j]
			if gm.Name != wm.Name || gm.Type != wm.Type || (wm.Value != "" && gm.Value != wm.Value) {
				t.Errorf("%q mod %d: %s %s %s, PoB %s %s %s", w.Line, j, gm.Name, gm.Type, gm.Value, wm.Name, wm.Type, wm.Value)
			}
			gt, wt := slices.Sorted(slices.Values(gm.Tags)), slices.Sorted(slices.Values(wm.Tags))
			if !slices.Equal(gt, wt) {
				t.Errorf("%q mod %d: tags %v, PoB %v", w.Line, j, gt, wt)
			}
		}
	}
	t.Logf("%d lines: %d parsed, %d not parsed by PoB", len(want), parsed, failed)
}

func TestParseModLinesLabels(t *testing.T) {
	initGameData(t)

	got := ParseModLines([]string{"", "  ", "# note", "^7# note", "not a mod at all", "+50 to maximum Life"})
	want := []ModLineStatus{ModLineBlank, ModLineBlank, ModLineComment, ModLineComment, ModLineUnsupported, ModLineParsed}
	for i := range want {
		if got[i].Status != want[i] {
			t.Errorf("line %d: %s, want %s", i, got[i].Status, want[i])
		}
	}
}

// TestConfigRelevance: ConfigRelevance keeps exactly the names the environment uses; a name it does not use must drop.
func TestConfigRelevance(t *testing.T) {
	initGameData(t)

	xml, err := os.ReadFile(filepath.Join("..", "..", "testdata", "corpus-"+raw.LatestVersion, "0037-ascendant-fire-trap-of-blasting-2.xml"))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	build, err := builds.ParseBuild(xml)
	if err != nil {
		t.Fatal(err)
	}
	env := calculator.NewCalculator(*build).BuildOutput(calculator.OutputModeMain)
	if env == nil {
		t.Fatal("no environment")
	}

	var anyCond string
	for name := range env.ConditionsUsed {
		anyCond = name
		break
	}
	got := env.ConfigRelevance(calculator.ConfigRelevance{
		Conds:              []string{anyCond, "NoSuchCondition"},
		Flags:              []string{"trap", "totem"},
		Skills:             []string{"Fire Trap", "Fireball"},
		TransfiguredSkills: []string{"Fire Trap", "Fireball"},
	})
	if !slices.Equal(got.Conds, []string{anyCond}) {
		t.Errorf("Conds = %v, want [%s]", got.Conds, anyCond)
	}
	if !slices.Equal(got.Flags, []string{"trap"}) {
		t.Errorf("Flags = %v, want [trap]", got.Flags)
	}
	usedFireTrap := false
	for s := range env.SkillsUsed {
		if strings.HasPrefix(s, "Fire Trap") {
			usedFireTrap = true
		}
	}
	if !usedFireTrap {
		t.Fatalf("REFUSING: the build uses no Fire Trap skill: %v", env.SkillsUsed)
	}
	if !slices.Equal(got.TransfiguredSkills, []string{"Fire Trap"}) {
		t.Errorf("TransfiguredSkills = %v, want [Fire Trap] (skills used: %v)", got.TransfiguredSkills, env.SkillsUsed)
	}
}

// TestConfigPlaceholdersAreTheCalculationsOwn: ConfigOptions.lua:1457-1589, 2246-2286.
func TestConfigPlaceholdersAreTheCalculationsOwn(t *testing.T) {
	initGameData(t)
	xml, err := os.ReadFile(filepath.Join("..", "..", "testdata", "corpus-"+raw.LatestVersion, "0280-elementalist-fireball.xml"))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	build, err := builds.ParseBuild(xml)
	if err != nil {
		t.Fatal(err)
	}
	pinnacle, meteor := "Pinnacle", "Sirus Meteor"
	build.SetConfigOption(pob.Input{Name: "enemyIsBoss", String: &pinnacle})
	build.SetConfigOption(pob.Input{Name: "presetBossSkills", String: &meteor})
	got := ConfigPlaceholders(build)
	env := calculator.NewCalculator(*build).BuildOutput(calculator.OutputModeMain)
	for _, k := range []string{"enemyLevel", "enemySpeed", "enemyPhysicalDamage", "enemyLightningResist", "enemyArmour"} {
		if got[k] == 0 || got[k] != env.ConfigPlaceholder[k] {
			t.Errorf("%s: page %v, calculation %v", k, got[k], env.ConfigPlaceholder[k])
		}
	}
	if got["enemyLevel"] != 84 {
		t.Errorf("a Pinnacle boss's level placeholder is %v", got["enemyLevel"])
	}
}
