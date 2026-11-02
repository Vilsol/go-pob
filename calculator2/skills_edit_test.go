package calculator2

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/pob2"
)

// skillsEditStep is one row of testdata/skills_edit_pob.jsonl, which testdata/skills_edit_oracle.lua writes from
// PoB-PoE2 for testdata/skills_edit.xml.
type skillsEditStep struct {
	Step            string             `json:"step"`
	MainSocketGroup int                `json:"mainSocketGroup"`
	Groups          []string           `json:"groups"`
	ActiveSkillSet  int                `json:"activeSkillSet"`
	Sets            []string           `json:"sets"`
	Group           int                `json:"group"`
	Text            string             `json:"text"`
	Errors          []string           `json:"errors"`
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

// ambiguousName is FindSkillGem's ambiguous name message.
var ambiguousName = regexp.MustCompile(`^(Ambiguous gem name '.*': matches )'(.*)', '(.*)'$`)

// sameErrMsg compares error messages; the two matches an ambiguous name reports are the first two in PoB-PoE2's
// pairs() order, so only the name is compared.
func sameErrMsg(got, want string) bool {
	g, w := ambiguousName.FindStringSubmatch(got), ambiguousName.FindStringSubmatch(want)
	if g == nil || w == nil {
		return got == want
	}
	return g[1] == w[1]
}

// copiedText is the oracle's last "copied" row, which the next paste pastes.
var copiedText string

// skillsPasteSteps are the oracle's "paste" scenario.
var skillsPasteSteps = map[string]skillsEdit{
	"Fire group": func(b *pob2.PathOfBuilding2, env *Environment) error {
		g := NewSocketGroup(b, env)
		return errors.Join(
			SetGem(b, env, g, 1, fireballID),
			SetGem(b, env, g, 2, elementalFocusID),
			SetGemLevel(b, env, g, 1, 15),
			SetGemQuality(b, env, g, 1, 20),
			SetGemCorruption(b, env, g, 1, 2),
			SetGemEnabled(b, env, g, 2, false),
			SetGemCount(b, env, g, 1, 3),
			SetSocketGroupLabel(b, env, g, "Fire"),
			SetSocketGroupWeaponSet(b, env, g, 2, false),
		)
	},
	"paste the Fire group": func(b *pob2.PathOfBuilding2, env *Environment) error {
		_, err := PasteSocketGroup(b, env, copiedText)
		return err
	},
	"paste a typed group": func(b *pob2.PathOfBuilding2, env *Environment) error {
		_, err := PasteSocketGroup(b, env, "Label: Typed\r\nWeapon Set: Set 2\r\nskillId:SparkPlayer Spark 5/10  2 C -1\r\n"+
			"Elemental Focus 1/0 DISABLED 1\r\nFrstbmb 3/0  1\r\nFrbl 2/0  1\r\nSpa 4/0  1\r\nfire 20/0  1\r\nNot A Gem 1/0  1\r\n")
		return err
	},
	"paste no gems": func(b *pob2.PathOfBuilding2, env *Environment) error {
		if index, err := PasteSocketGroup(b, env, "nothing to paste"); err != nil || index != 0 {
			return fmt.Errorf("pasted a group with no gems at %d: %w", index, err)
		}
		return nil
	},
}

// skillsUndoSteps are the oracle's "undo" scenario: every edit adds an undo state, as the Skills tab's controls do.
func skillsUndoSteps() map[string]skillsEdit {
	var h *SkillsUndo
	edit := func(f skillsEdit) skillsEdit {
		return func(b *pob2.PathOfBuilding2, env *Environment) error {
			if h == nil {
				h = &SkillsUndo{}
				h.ResetUndo(b)
			}
			if err := f(b, env); err != nil {
				return err
			}
			h.AddUndoState(b, false)
			return nil
		}
	}
	undo := func(redo bool, times int) skillsEdit {
		return func(b *pob2.PathOfBuilding2, _ *Environment) error {
			for range times {
				if redo {
					h.Redo(b)
				} else {
					h.Undo(b)
				}
			}
			return nil
		}
	}
	newGroup := edit(func(b *pob2.PathOfBuilding2, env *Environment) error { NewSocketGroup(b, env); return nil })
	return map[string]skillsEdit{
		"new group":             newGroup,
		"add Fireball":          edit(func(b *pob2.PathOfBuilding2, env *Environment) error { return SetGem(b, env, 2, 1, fireballID) }),
		"label Fire":            edit(func(b *pob2.PathOfBuilding2, env *Environment) error { return SetSocketGroupLabel(b, env, 2, "Fire") }),
		"undo the label":        undo(false, 1),
		"undo Fireball":         undo(false, 1),
		"redo Fireball":         undo(true, 1),
		"new set Second":        edit(func(b *pob2.PathOfBuilding2, env *Environment) error { NewSkillSet(b, env, "Second"); return nil }),
		"undo the new set":      undo(false, 1),
		"undo Fireball again":   undo(false, 1),
		"new group clears redo": newGroup,
		"redo nothing": func(b *pob2.PathOfBuilding2, _ *Environment) error {
			if h.Redo(b) {
				return errors.New("a new edit left a redo")
			}
			return nil
		},
		"undo past the loaded state": undo(false, 3),
	}
}

// setAt is the id of the skill set at a place in the Manage list.
func setAt(b *pob2.PathOfBuilding2, i int) int { return b.Skills.SkillSets[i-1].ID }

// skillSetsSteps are the oracle's "sets" scenario.
var skillSetsSteps = map[string]skillsEdit{
	"new set Second": func(b *pob2.PathOfBuilding2, env *Environment) error { NewSkillSet(b, env, "Second"); return nil },
	"add Fireball to Second": func(b *pob2.PathOfBuilding2, env *Environment) error {
		return SetGem(b, env, NewSocketGroup(b, env), 1, fireballID)
	},
	"copy set 1": func(b *pob2.PathOfBuilding2, env *Environment) error {
		_, err := CopySkillSet(b, env, setAt(b, 1), "Copy")
		return err
	},
	"rename set 2 Fire": func(b *pob2.PathOfBuilding2, _ *Environment) error { return RenameSkillSet(b, setAt(b, 2), "Fire") },
	"switch to set 2": func(b *pob2.PathOfBuilding2, env *Environment) error {
		SetActiveSkillSet(b, env, setAt(b, 2))
		return nil
	},
	"delete active set 2": func(b *pob2.PathOfBuilding2, env *Environment) error { return DeleteSkillSet(b, env, setAt(b, 2)) },
	"delete active set 1": func(b *pob2.PathOfBuilding2, env *Environment) error { return DeleteSkillSet(b, env, setAt(b, 1)) },
	"new set Third":       func(b *pob2.PathOfBuilding2, env *Environment) error { NewSkillSet(b, env, "Third"); return nil },
	"new set Fourth":      func(b *pob2.PathOfBuilding2, env *Environment) error { NewSkillSet(b, env, "Fourth"); return nil },
	"switch to set 1": func(b *pob2.PathOfBuilding2, env *Environment) error {
		SetActiveSkillSet(b, env, setAt(b, 1))
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
	out.Skills.ActiveSkillSet = b.Skills.ActiveSkillSet
	for _, set := range b.Skills.SkillSets {
		set.Unparsed = pob2.Unparsed{}
		set.Skills = slices.Clone(set.Skills)
		for i := range set.Skills {
			set.Skills[i].Unparsed = pob2.Unparsed{}
			set.Skills[i].Gems = slices.Clone(set.Skills[i].Gems)
			asLoaded(&set.Skills[i])
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

// asLoaded fills in the defaults LoadSkill gives a group saved with "nil", which PoB-PoE2 then saves; a loaded
// group keeps what its file says here (SkillsTab.lua:294-364).
func asLoaded(s *pob2.Skill) {
	if s.IncludeInFullDPS.IsNil {
		s.IncludeInFullDPS = pob2.Nil[bool]{}
	}
	if s.MainActiveSkillCalcs.IsNil {
		s.MainActiveSkillCalcs = pob2.Nil[int]{V: 1}
	}
	for j := range s.Gems {
		g := &s.Gems[j]
		if g.Corrupted.IsNil {
			g.Corrupted = pob2.Nil[bool]{}
		}
		if g.StatSetIndex == nil {
			g.StatSetIndex = &pob2.Nil[int]{IsNil: true}
		}
		if g.StatSetIndexCalcs == nil {
			g.StatSetIndexCalcs = &pob2.Nil[int]{IsNil: true}
		}
	}
}

// Every edit recalculates; the sidebar's numbers, the socket group labels, the main group and the saved skill
// sets are PoB-PoE2's for the same edits through its Skills tab controls.
func TestSkillsEditMatchesPoB(t *testing.T) {
	for _, scenario := range skillsEditScenarios {
		t.Run(scenario.oracle, func(t *testing.T) { runSkillsEdits(t, scenario.fixture, scenario.oracle, scenario.steps) })
	}
}

// skillsEditScenarios are the oracle's scenarios: its argument, the build it loads and the rows it writes.
var skillsEditScenarios = []struct {
	name, fixture, oracle string
	steps                 map[string]skillsEdit
}{
	{"edits", "skills_edit", "skills_edit", skillsEditSteps},
	{"generated", "skills_edit_generated", "skills_edit_generated", skillsGeneratedSteps},
	{"sets", "skills_edit", "skills_sets", skillSetsSteps},
	{"sets", "skills_sets_legacy", "skills_sets_legacy", skillSetsSteps},
	{"paste", "skills_edit", "skills_paste", skillsPasteSteps},
	{"undo", "skills_edit", "skills_undo", skillsUndoSteps()},
}

func runSkillsEdits(t *testing.T, fixture, oracle string, steps map[string]skillsEdit) {
	t.Helper()
	rows := readSkillsEditOracle(t, filepath.Join("testdata", oracle+"_pob.jsonl"))
	raw, err := os.ReadFile(filepath.Join("testdata", fixture+".xml"))
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
		if row.Step == "errors" {
			gems := env.SocketGroupList[row.Group-1].GemList
			if len(gems) != len(row.Errors) {
				t.Fatalf("group %d has %d gems, PoB-PoE2 %d", row.Group, len(gems), len(row.Errors))
			}
			for i, gem := range gems {
				if got := GemErrMsg(gem); !sameErrMsg(got, row.Errors[i]) {
					t.Errorf("gem %d error %q, PoB-PoE2 %q", i+1, got, row.Errors[i])
				}
			}
			continue
		}
		if row.Step == "copied" {
			copiedText = row.Text
			if got, err := CopySocketGroup(env, row.Group); err != nil || got != row.Text {
				t.Errorf("group %d copies %q (%v), PoB-PoE2 %q", row.Group, got, err, row.Text)
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
		if row.Sets != nil {
			views, active := SkillSetList(build)
			sets := make([]string, 0, len(views))
			for _, set := range views {
				sets = append(sets, fmt.Sprintf("%d:%s", set.ID, set.Title))
			}
			if active != row.ActiveSkillSet || !slices.Equal(sets, row.Sets) {
				t.Errorf("%s: skill sets %q, active %d; PoB-PoE2 %q, active %d", row.Step, sets, active, row.Sets, row.ActiveSkillSet)
			}
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
	for _, scenario := range skillsEditScenarios {
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
		want, err := os.ReadFile(filepath.Join("testdata", scenario.oracle+"_pob.jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("testdata/%s_pob.jsonl is stale; PoB-PoE2 now writes:\n%s", scenario.oracle, got)
		}
	}
}
