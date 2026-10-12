package calculator

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/pob"
)

const sidebarControlsExpected = "../testdata/oracle/sidebar_controls_expected.jsonl"

type sidebarControlsCase struct {
	File               string   `json:"file"`
	Used               int      `json:"used"`
	AscUsed            int      `json:"ascUsed"`
	SecondaryAscUsed   int      `json:"secondaryAscUsed"`
	Sockets            int      `json:"sockets"`
	ExtraPoints        float64  `json:"extraPoints"`
	Skills             []string `json:"skills"`
	SkillEnabled       bool     `json:"skillEnabled"`
	MainActiveSkill    int      `json:"mainActiveSkill"`
	PartShown          bool     `json:"partShown"`
	Parts              []string `json:"parts"`
	SkillPart          int      `json:"skillPart"`
	StageShown         bool     `json:"stageShown"`
	StageCount         string   `json:"stageCount"`
	MineShown          bool     `json:"mineShown"`
	MineCount          string   `json:"mineCount"`
	MinionShown        bool     `json:"minionShown"`
	MinionEnabled      bool     `json:"minionEnabled"`
	MinionLibraryShown bool     `json:"minionLibraryShown"`
	Minions            []string `json:"minions"`
	Minion             int      `json:"minion"`
	MinionSkillShown   bool     `json:"minionSkillShown"`
	MinionSkillEnabled bool     `json:"minionSkillEnabled"`
	MinionSkills       []string `json:"minionSkills"`
	MinionSkill        int      `json:"minionSkill"`
}

func loadSidebarControlsCases(t *testing.T) []sidebarControlsCase {
	t.Helper()
	f, err := os.Open(sidebarControlsExpected)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var cases []sidebarControlsCase
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		var c sidebarControlsCase
		if err := json.Unmarshal(scanner.Bytes(), &c); err != nil {
			t.Fatal(err)
		}
		cases = append(cases, c)
	}
	if len(cases) == 0 {
		t.Fatal("no cases in " + sidebarControlsExpected)
	}
	return cases
}

func sidebarEnv(t *testing.T, file string) *Environment {
	t.Helper()
	raw, err := os.ReadFile("../testdata/corpus-3.29/" + file)
	if err != nil {
		t.Fatal(err)
	}
	build, err := builds.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	return BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestMainSkillControlsAgainstPoB(t *testing.T) {
	for _, want := range loadSidebarControlsCases(t) {
		t.Run(want.File, func(t *testing.T) {
			got := sidebarEnv(t, want.File).MainSkillControls()
			if got == nil {
				t.Fatal("no controls")
			}
			minions := make([]string, 0, len(got.Minions))
			for _, m := range got.Minions {
				minions = append(minions, m.Label)
			}
			check := func(name string, ok bool, g, w any) {
				if !ok {
					t.Errorf("%s: got %v, PoB %v", name, g, w)
				}
			}
			check("skills", sameStrings(got.Skills, want.Skills), got.Skills, want.Skills)
			check("skillEnabled", got.SkillEnabled == want.SkillEnabled, got.SkillEnabled, want.SkillEnabled)
			check("mainActiveSkill", got.MainActiveSkill == want.MainActiveSkill, got.MainActiveSkill, want.MainActiveSkill)
			check("partShown", got.PartShown == want.PartShown, got.PartShown, want.PartShown)
			check("parts", sameStrings(got.Parts, want.Parts), got.Parts, want.Parts)
			if want.PartShown {
				check("skillPart", got.SkillPart == want.SkillPart, got.SkillPart, want.SkillPart)
			}
			check("stageShown", got.StageShown == want.StageShown, got.StageShown, want.StageShown)
			check("stageCount", got.StageCount == want.StageCount, got.StageCount, want.StageCount)
			check("mineShown", got.MineShown == want.MineShown, got.MineShown, want.MineShown)
			check("mineCount", got.MineCount == want.MineCount, got.MineCount, want.MineCount)
			check("minionShown", got.MinionShown == want.MinionShown, got.MinionShown, want.MinionShown)
			check("minionEnabled", got.MinionEnabled == want.MinionEnabled, got.MinionEnabled, want.MinionEnabled)
			check("minionLibraryShown", got.MinionLibraryShown == want.MinionLibraryShown, got.MinionLibraryShown, want.MinionLibraryShown)
			check("minions", sameStrings(minions, want.Minions), minions, want.Minions)
			if want.MinionShown {
				check("minion", got.Minion == want.Minion, got.Minion, want.Minion)
			}
			check("minionSkillShown", got.MinionSkillShown == want.MinionSkillShown, got.MinionSkillShown, want.MinionSkillShown)
			check("minionSkillEnabled", got.MinionSkillEnabled == want.MinionSkillEnabled, got.MinionSkillEnabled, want.MinionSkillEnabled)
			check("minionSkills", sameStrings(got.MinionSkills, want.MinionSkills), got.MinionSkills, want.MinionSkills)
			if want.MinionSkillShown {
				check("minionSkill", got.MinionSkill == want.MinionSkill, got.MinionSkill, want.MinionSkill)
			}
		})
	}
}

func TestCountAllocNodesAgainstPoB(t *testing.T) {
	for _, want := range loadSidebarControlsCases(t) {
		t.Run(want.File, func(t *testing.T) {
			used, ascUsed, secondaryAscUsed, sockets := sidebarEnv(t, want.File).CountAllocNodes()
			if used != want.Used || ascUsed != want.AscUsed || secondaryAscUsed != want.SecondaryAscUsed || sockets != want.Sockets {
				t.Errorf("got %d/%d/%d/%d, PoB %d/%d/%d/%d", used, ascUsed, secondaryAscUsed, sockets,
					want.Used, want.AscUsed, want.SecondaryAscUsed, want.Sockets)
			}
		})
	}
}

// TestMainSkillControlsCarryTheDisableReason: Build.lua:1868-1871: a main skill the gear cannot use carries skillFlags.disable and its disableReason to the stat box.
func TestMainSkillControlsCarryTheDisableReason(t *testing.T) {
	raw, err := os.ReadFile("../testdata/corpus-3.29/0008-necromancer-vaal-summon-skeletons.xml")
	if err != nil {
		t.Fatal(err)
	}
	build, err := builds.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
	if c := env.MainSkillControls(); c == nil || c.SkillDisabled {
		t.Fatalf("control: the build's own main skill is disabled: %+v", c)
	}
	set := &build.Skills.SkillSets[build.ActiveSkillSetIndex()]
	main := &set.Skills[env.MainSocketGroup]
	main.Gems = []pob.Gem{{GemID: "Metadata/Items/Gems/SkillGemTornadoShot", SkillID: "TornadoShot", NameSpec: "Tornado Shot", Level: 20, Enabled: true, EnableGlobal1: true}}
	main.MainActiveSkill = 1
	env = BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
	c := env.MainSkillControls()
	if c == nil || !c.SkillDisabled || c.DisableReason != "No usable weapon equipped" {
		t.Fatalf("Tornado Shot without a bow: %+v", c)
	}
}
