package calculator2

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/pob2"
)

// skillsEditStep is one row of testdata/skills_edit_pob.jsonl, which testdata/skills_edit_oracle.lua writes from
// PoB-PoE2 for testdata/skills_edit.xml.
type skillsEditStep struct {
	Step            string             `json:"step"`
	MainSocketGroup int                `json:"mainSocketGroup"`
	Groups          []string           `json:"groups"`
	Output          map[string]float64 `json:"output"`
	XML             string             `json:"xml"`
}

const (
	fireballID       = "Metadata/Items/Gems/SkillGemFireball"
	elementalFocusID = "Metadata/Items/Gems/SkillGemElementalFocusSupport"
	frostBombID      = "Metadata/Items/Gems/SkillGemFrostBomb"
	rollingSlamID    = "Metadata/Items/Gems/SkillGemRollingSlam"
)

type skillsEdit = func(b *pob2.PathOfBuilding2, env *Environment) error

// skillsEditSteps are the oracle's "edits" scenario.
var skillsEditSteps = map[string]skillsEdit{
	"new group": func(b *pob2.PathOfBuilding2, env *Environment) error { NewSocketGroup(b, env); return nil },
	"main group 2": func(b *pob2.PathOfBuilding2, _ *Environment) error {
		SetMainSocketGroup(b, 2)
		return nil
	},
	"add Fireball":            func(b *pob2.PathOfBuilding2, env *Environment) error { return SetGem(b, env, 2, 1, fireballID) },
	"add Elemental Focus":     func(b *pob2.PathOfBuilding2, env *Environment) error { return SetGem(b, env, 2, 2, elementalFocusID) },
	"Fireball level 15":       func(b *pob2.PathOfBuilding2, env *Environment) error { return SetGemLevel(b, env, 2, 1, 15) },
	"Fireball quality 20":     func(b *pob2.PathOfBuilding2, env *Environment) error { return SetGemQuality(b, env, 2, 1, 20) },
	"disable Elemental Focus": func(b *pob2.PathOfBuilding2, env *Environment) error { return SetGemEnabled(b, env, 2, 2, false) },
	"enable Elemental Focus":  func(b *pob2.PathOfBuilding2, env *Environment) error { return SetGemEnabled(b, env, 2, 2, true) },
	"Fireball count 3":        func(b *pob2.PathOfBuilding2, env *Environment) error { return SetGemCount(b, env, 2, 1, 3) },
	"label Fire":              func(b *pob2.PathOfBuilding2, env *Environment) error { return SetSocketGroupLabel(b, env, 2, "Fire") },
	"include in Full DPS": func(b *pob2.PathOfBuilding2, env *Environment) error {
		return SetSocketGroupIncludeInFullDPS(b, env, 2, true)
	},
	"disable group":     func(b *pob2.PathOfBuilding2, env *Environment) error { return SetSocketGroupEnabled(b, env, 2, false) },
	"enable group":      func(b *pob2.PathOfBuilding2, env *Environment) error { return SetSocketGroupEnabled(b, env, 2, true) },
	"move group 2 to 1": func(b *pob2.PathOfBuilding2, env *Environment) error { return MoveSocketGroup(b, env, 2, 1) },
	"replace Elemental Focus with Frost Bomb": func(b *pob2.PathOfBuilding2, env *Environment) error {
		return SetGem(b, env, 1, 2, frostBombID)
	},
	"delete Fireball": func(b *pob2.PathOfBuilding2, env *Environment) error { return RemoveGem(b, env, 1, 1) },
	"delete group 2":  func(b *pob2.PathOfBuilding2, env *Environment) error { return DeleteSocketGroup(b, env, 2) },
}

// skillsGeneratedSteps are the oracle's "generated" scenario: the warrior's maces grant a default attack group.
var skillsGeneratedSteps = map[string]skillsEdit{
	"new group":        func(b *pob2.PathOfBuilding2, env *Environment) error { NewSocketGroup(b, env); return nil },
	"add Rolling Slam": func(b *pob2.PathOfBuilding2, env *Environment) error { return SetGem(b, env, 3, 1, rollingSlamID) },
	"main group new": func(b *pob2.PathOfBuilding2, _ *Environment) error {
		SetMainSocketGroup(b, 3)
		return nil
	},
	"Rolling Slam +1 level corruption": func(b *pob2.PathOfBuilding2, env *Environment) error { return SetGemCorruption(b, env, 3, 1, 2) },
	"new group set 1 off": func(b *pob2.PathOfBuilding2, env *Environment) error {
		return SetSocketGroupWeaponSet(b, env, 3, 1, false)
	},
	"new group set 2 off refused": func(b *pob2.PathOfBuilding2, env *Environment) error {
		return SetSocketGroupWeaponSet(b, env, 3, 2, false)
	},
	"support group 2": func(b *pob2.PathOfBuilding2, env *Environment) error { return SetGem(b, env, 2, 2, elementalFocusID) },
	"group 2 count 2": func(b *pob2.PathOfBuilding2, env *Environment) error { return SetSocketGroupCount(b, env, 2, 2) },
	"disable group 2": func(b *pob2.PathOfBuilding2, env *Environment) error { return SetSocketGroupEnabled(b, env, 2, false) },
	"delete all": func(b *pob2.PathOfBuilding2, _ *Environment) error {
		DeleteAllSocketGroups(b)
		return nil
	},
}

func readSkillsEditOracle(t *testing.T, path string) []skillsEditStep {
	t.Helper()
	raw, err := os.ReadFile(path) //nolint:gosec // test fixture
	if err != nil {
		t.Fatal(err)
	}
	var rows []skillsEditStep
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var row skillsEditStep
		if err := json.Unmarshal(sc.Bytes(), &row); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	return rows
}

// skillSetsXML is the build's skill sets and main group as the build file writes them, without what a load keeps
// for a round trip.
func skillSetsXML(t *testing.T, b *pob2.PathOfBuilding2) string {
	t.Helper()
	out := &pob2.PathOfBuilding2{}
	out.Build.MainSocketGroup = b.Build.MainSocketGroup
	for _, set := range b.Skills.SkillSets {
		set.Unparsed = pob2.Unparsed{}
		for i := range set.Skills {
			set.Skills[i].Unparsed = pob2.Unparsed{}
			for j := range set.Skills[i].Gems {
				set.Skills[i].Gems[j].Unparsed = pob2.Unparsed{}
			}
		}
		out.Skills.SkillSets = append(out.Skills.SkillSets, set)
	}
	text, err := pob2.SerializeBuild(out)
	if err != nil {
		t.Fatal(err)
	}
	return string(text)
}

// Every edit recalculates; the sidebar's numbers, the socket group labels, the main group and the saved skill
// sets are PoB-PoE2's for the same edits through its Skills tab controls.
func TestSkillsEditMatchesPoB(t *testing.T) {
	t.Run("edits", func(t *testing.T) { runSkillsEdits(t, "skills_edit", skillsEditSteps) })
	t.Run("generated", func(t *testing.T) { runSkillsEdits(t, "skills_edit_generated", skillsGeneratedSteps) })
}

func runSkillsEdits(t *testing.T, name string, steps map[string]skillsEdit) {
	t.Helper()
	rows := readSkillsEditOracle(t, filepath.Join("testdata", name+"_pob.jsonl"))
	raw, err := os.ReadFile(filepath.Join("testdata", name+".xml"))
	if err != nil {
		t.Fatal(err)
	}
	build, err := pob2.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	tree := loadTestTree(t)
	var env *Environment
	for _, row := range rows {
		if row.Step == "saved" {
			SyncSocketGroups(build, env)
			saved, err := pob2.ParseBuild([]byte(row.XML))
			if err != nil {
				t.Fatal(err)
			}
			if got, want := skillSetsXML(t, build), skillSetsXML(t, saved); got != want {
				t.Errorf("saved skills:\n%s\nPoB-PoE2:\n%s", got, want)
			}
			continue
		}
		if row.Step != "load" {
			edit := steps[row.Step]
			if edit == nil {
				t.Fatalf("no edit for step %q", row.Step)
			}
			if err := edit(build, env); err != nil {
				t.Fatalf("%s: %v", row.Step, err)
			}
		}
		env, err = BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
		if err != nil {
			t.Fatalf("%s: %v", row.Step, err)
		}
		if env.MainSocketGroup != row.MainSocketGroup {
			t.Errorf("%s: main socket group %d, PoB-PoE2 %d", row.Step, env.MainSocketGroup, row.MainSocketGroup)
		}
		views := SocketGroupViews(env)
		labels := make([]string, len(views))
		for i, v := range views {
			labels[i] = v.Label
		}
		if got, want := len(labels), len(row.Groups); got != want {
			t.Errorf("%s: groups %q, PoB-PoE2 %q", row.Step, labels, row.Groups)
		} else {
			for i := range labels {
				if labels[i] != row.Groups[i] {
					t.Errorf("%s: groups %q, PoB-PoE2 %q", row.Step, labels, row.Groups)
					break
				}
			}
		}
		for stat, want := range row.Output {
			if got := env.Player.Output[stat]; math.Abs(got-want) > 1e-9*math.Max(1, math.Abs(want)) {
				t.Errorf("%s: %s %v, PoB-PoE2 %v", row.Step, stat, got, want)
			}
		}
	}
}

// The committed oracle rows are what testdata/skills_edit_oracle.lua writes from the pinned PoB-PoE2; CORPUS=true
// runs it, as it starts PoB-PoE2.
func TestSkillsEditOracleIsCurrent(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true runs PoB-PoE2")
	}
	pob, luajit := pobPoE2Probe(t)
	probe, err := filepath.Abs(filepath.Join("testdata", "skills_edit_oracle.lua"))
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []struct{ name, fixture string }{{"edits", "skills_edit"}, {"generated", "skills_edit_generated"}} {
		fixture, err := filepath.Abs(filepath.Join("testdata", scenario.fixture+".xml"))
		if err != nil {
			t.Fatal(err)
		}
		out := filepath.Join(t.TempDir(), "skills.jsonl")
		cmd := exec.CommandContext(context.Background(), "setarch", "-R", luajit, probe, out, fixture, scenario.name)
		cmd.Dir = filepath.Join(pob, "src")
		cmd.Env = append(os.Environ(),
			"LUA_PATH="+pob+"/runtime/lua/?.lua;"+pob+"/runtime/lua/?/init.lua;"+pob+"/.luarocks/share/lua/5.1/?.lua;"+pob+"/.luarocks/share/lua/5.1/?/init.lua;;",
			"LUA_CPATH="+pob+"/.luarocks/lib/lua/5.1/?.so;;")
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("the skills oracle failed: %v\n%s", err, stderr.String())
		}
		got, err := os.ReadFile(out) //nolint:gosec // the test's own temp dir
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(filepath.Join("testdata", scenario.fixture+"_pob.jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("testdata/%s_pob.jsonl is stale; PoB-PoE2 now writes:\n%s", scenario.fixture, got)
		}
	}
}
