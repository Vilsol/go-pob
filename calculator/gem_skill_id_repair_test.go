package calculator

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
)

// TestLoadRepairsGemIDInSkillID: the fixture was saved with skillId = gem id on every picked gem.
// The pinned PoB loads it with those gems missing (minion TotalDPS 313816.35) and loads the
// re-saved build with them (1891483.0676892).
func TestLoadRepairsGemIDInSkillID(t *testing.T) {
	requireGameData(t, "3.29")
	file, err := os.ReadFile("testdata/item-granted-supports/dancing-dervish.xml")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(file, []byte(`skillId="Metadata/Items/Gems/SupportGemMinionDamage"`)) {
		t.Fatal("the fixture no longer carries a gem id in skillId")
	}
	build, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatal(err)
	}
	out, err := builds.SerializeBuild(build)
	if err != nil {
		t.Fatal(err)
	}
	if n := bytes.Count(out, []byte(`skillId="Metadata/`)); n != 0 {
		t.Errorf("%d gems are still saved with a gem id in skillId", n)
	}
	for _, want := range []string{"SupportMinionDamage", "SupportPredator", "Despair", "SummonStoneGolem", "SupportGenerosity", "ManifestDancingDervishes"} {
		if !bytes.Contains(out, []byte(`skillId="`+want+`"`)) {
			t.Errorf("saved build lacks skillId=%q", want)
		}
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
	if got, want := env.Minion.Output["TotalDPS"], 1891483.0676892; math.Abs(got-want) > 1e-6*want {
		t.Errorf("minion TotalDPS %v, PoB on the repaired build %v", got, want)
	}
}

// TestGemIDsNeverCollideWithGrantedEffectIDs: the repair assumes a gem id is never also a skill id.
func TestGemIDsNeverCollideWithGrantedEffectIDs(t *testing.T) {
	effects := make(map[string]bool, len(raw.Gems)+len(raw.GemVariants))
	for _, row := range raw.Gems {
		effects[row.GrantedEffectID] = true
		effects[row.SecondaryGrantedEffectID] = true
	}
	for id, row := range raw.GemVariants {
		effects[id] = true
		effects[row.GrantedEffectID] = true
		effects[row.SecondaryGrantedEffectID] = true
	}
	for id := range raw.Gems {
		if effects[id] {
			t.Errorf("gem id %q is also a granted effect id", id)
		}
	}
}

var corpusGemTag = regexp.MustCompile(`<Gem [^>]*>`)
var corpusGemAttr = regexp.MustCompile(`\s(gemId|skillId)="([^"]*)"`)

// TestLoadKeepsCorrectSkillIDs: corpus builds come from PoB, so loading must not change a skillId.
func TestLoadKeepsCorrectSkillIDs(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true walks the corpus")
	}
	dir, names := corpusNamesForLoadedData(t)
	gems, repairable := 0, 0
	for i, name := range names {
		file, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		var saved []string
		for _, tag := range corpusGemTag.FindAll(file, -1) {
			gemID, skillID := "", ""
			for _, m := range corpusGemAttr.FindAllSubmatch(tag, -1) {
				if string(m[1]) == "skillId" {
					skillID = string(m[2])
				} else {
					gemID = string(m[2])
				}
			}
			if _, ok := raw.Gems[skillID]; ok && (gemID == "" || gemID == skillID) {
				repairable++
				t.Errorf("%s: PoB saved skillId %q with gemId %q", name, skillID, gemID)
			}
			saved = append(saved, skillID)
		}
		gems += len(saved)
		if i%25 != 0 {
			continue
		}
		build, err := builds.ParseBuild(file)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var loaded []string
		for _, set := range build.Skills.SkillSets {
			for _, skill := range set.Skills {
				for _, gem := range skill.Gems {
					loaded = append(loaded, gem.SkillID)
				}
			}
		}
		if strings.Join(loaded, "\n") != strings.Join(saved, "\n") {
			t.Errorf("%s: skillIds changed on load", name)
		}
	}
	if gems == 0 {
		t.Fatal("REFUSING: the corpus holds no <Gem> elements")
	}
	t.Logf("%d builds, %d gems, %d the repair would touch, every 25th build parsed", len(names), gems, repairable)
}
