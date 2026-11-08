package exposition

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// treeSpecsRow is one row of testdata/tree_specs_<scenario>_pob.jsonl, which testdata/tree_specs_oracle.lua writes
// from the pinned PoB-PoE2.
type treeSpecsRow struct {
	Step            string              `json:"step"`
	Active          int                 `json:"active"`
	ClassName       string              `json:"className"`
	AscendClassName string              `json:"ascendClassName"`
	Labels          []string            `json:"labels"`
	Specs           []map[string]string `json:"specs"`
	Output          map[string]float64  `json:"output"`
}

// oracleSpec is the idx-th <Spec> as the oracle prints PassiveSpec:Save's output of it.
func oracleSpec(t *testing.T, c *Calculator, idx int) map[string]string {
	t.Helper()
	s := &c.build.Tree.Specs[idx-1]
	spec, err := c.specAt(idx)
	mustOK(t, err)
	save := spec.Save()
	out := map[string]string{
		"title": specTitle(s), "treeVersion": save.TreeVersion, "classId": strconv.Itoa(save.ClassID), "ascendClassId": strconv.Itoa(save.AscendClassID),
		"nodes": save.Nodes, "masteryEffects": save.MasteryEffects, "secondaryAscendClassId": "nil",
		"overrides": save.StrNodes + "|" + save.DexNodes + "|" + save.IntNodes,
	}
	if s.Title == nil {
		out["title"] = ""
	}
	if !s.SecondaryAscendClassID.IsNil {
		out["secondaryAscendClassId"] = strconv.Itoa(s.SecondaryAscendClassID.V)
	}
	for set, nodes := range save.WeaponSets {
		out["WeaponSet"+strconv.Itoa(set)] = nodes
	}
	socks := make([]string, 0, len(save.Sockets))
	for _, sock := range save.Sockets {
		socks = append(socks, sock.NodeID+":"+sock.ItemID)
	}
	out["sockets"] = strings.Join(socks, ",")
	return out
}

// treeSpecsSteps are the oracle's steps, by its step names.
var treeSpecsSteps = map[string]func(t *testing.T, c *Calculator){
	"new Second":         func(t *testing.T, c *Calculator) { mustOK(t, c.NewTreeSpec("Second")) },
	"copy 1 Radius copy": func(t *testing.T, c *Calculator) { mustOK(t, c.CopyTreeSpec(1, "Radius copy")) },
	"rename 2 Renamed":   func(_ *testing.T, c *Calculator) { c.RenameTreeSpec(2, "Renamed") },
	"select 4":           func(_ *testing.T, c *Calculator) { c.SetActiveTreeSpec(4) },
	"move 4 1":           func(_ *testing.T, c *Calculator) { c.MoveTreeSpec(4, 1) },
	"move 2 3":           func(_ *testing.T, c *Calculator) { c.MoveTreeSpec(2, 3) },
	"delete 3":           func(_ *testing.T, c *Calculator) { c.DeleteTreeSpec(3) },
	"delete 1":           func(_ *testing.T, c *Calculator) { c.DeleteTreeSpec(1) },
	"select 2":           func(_ *testing.T, c *Calculator) { c.SetActiveTreeSpec(2) },
	"new Third":          func(t *testing.T, c *Calculator) { mustOK(t, c.NewTreeSpec("Third")) },
	"delete 2":           func(_ *testing.T, c *Calculator) { c.DeleteTreeSpec(2) },
	"ascendancy 1":       func(t *testing.T, c *Calculator) { mustOK(t, c.ChangeTreeAscendancy(1)) },
	"ascendancy 0":       func(t *testing.T, c *Calculator) { mustOK(t, c.ChangeTreeAscendancy(0)) },
	"class Sorceress cancel": func(t *testing.T, c *Calculator) {
		if ask, err := c.ChangeTreeClass(classIDOf(t, "Sorceress"), TreeClassAsk); err != nil || !ask {
			t.Fatalf("the change does not ask first: %v %v", ask, err)
		}
	},
	"class Sorceress connect": func(t *testing.T, c *Calculator) { changeClass(t, c, "Sorceress", TreeClassConnect) },
	"class Ranger continue":   func(t *testing.T, c *Calculator) { changeClass(t, c, "Ranger", TreeClassContinue) },
	"class Witch": func(t *testing.T, c *Calculator) {
		if ask, err := c.ChangeTreeClass(classIDOf(t, "Witch"), TreeClassAsk); err != nil || ask {
			t.Fatalf("an empty tree's class change asks: %v %v", ask, err)
		}
	},
}

func mustOK(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func classIDOf(t *testing.T, name string) int {
	t.Helper()
	list, err := TreeClassList()
	mustOK(t, err)
	for _, c := range list {
		if c.Label == name {
			return c.ClassID
		}
	}
	t.Fatalf("no class %s", name)
	return 0
}

func changeClass(t *testing.T, c *Calculator, name string, choice int) {
	t.Helper()
	ask, err := c.ChangeTreeClass(classIDOf(t, name), choice)
	mustOK(t, err)
	if ask {
		t.Fatalf("an answered class change asks again")
	}
}

func readTreeSpecsOracle(t *testing.T, scenario string) []treeSpecsRow {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "tree_specs_"+scenario+"_pob.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var rows []treeSpecsRow
	sc := bufio.NewScanner(f)
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		var r treeSpecsRow
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, r)
	}
	return rows
}

func TestTreeSpecsMatchPoB(t *testing.T) {
	for _, scenario := range []string{"specs", "class"} {
		t.Run(scenario, func(t *testing.T) {
			calc, _ := openFixture(t, "tree_specs.xml")
			for _, want := range readTreeSpecsOracle(t, scenario) {
				if step, ok := treeSpecsSteps[want.Step]; ok {
					step(t, calc)
				} else if want.Step != "load" {
					t.Fatalf("no step %q", want.Step)
				}
				out, err := calc.Calculate()
				mustOK(t, err)
				view := calc.TreeSpecs()
				got := treeSpecsRow{Step: want.Step, Active: view.Active, ClassName: out.ClassName, AscendClassName: out.AscendClassName, Labels: view.Labels}
				for i := range calc.build.Tree.Specs {
					got.Specs = append(got.Specs, oracleSpec(t, calc, i+1))
				}
				got.Output = map[string]float64{}
				for stat := range want.Output {
					got.Output[stat] = math.Round(out.Output[stat])
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("%s:\n got %+v\nwant %+v", want.Step, got, want)
				}
			}
		})
	}
}

// The committed oracle rows are what testdata/tree_specs_oracle.lua writes from the pinned PoB-PoE2; CORPUS=true
// runs it, as it starts PoB-PoE2.
func TestTreeSpecsOracleIsCurrent(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true runs PoB-PoE2")
	}
	common, err := exec.CommandContext(context.Background(), "git", "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err != nil {
		t.Skipf("SKIP: not in a git checkout: %v", err)
	}
	main := filepath.Dir(strings.TrimSpace(string(common)))
	pob := os.Getenv("POB2_DIR")
	if pob == "" {
		pob = filepath.Join(main, "..", "PathOfBuilding-PoE2")
	}
	luajit := filepath.Join(main, ".luajit-det", "bin", "luajit")
	for _, p := range []string{filepath.Join(pob, "src", "HeadlessWrapper.lua"), luajit} {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("SKIP: %s is absent", p)
		}
	}
	probe, err := filepath.Abs(filepath.Join("testdata", "tree_specs_oracle.lua"))
	mustOK(t, err)
	fixture, err := filepath.Abs(filepath.Join("testdata", "tree_specs.xml"))
	mustOK(t, err)
	for _, scenario := range []string{"specs", "class"} {
		out := filepath.Join(t.TempDir(), "tree.jsonl")
		cmd := exec.CommandContext(context.Background(), "setarch", "-R", luajit, probe, out, fixture, scenario)
		cmd.Dir = filepath.Join(pob, "src")
		cmd.Env = append(os.Environ(),
			"LUA_PATH="+pob+"/runtime/lua/?.lua;"+pob+"/runtime/lua/?/init.lua;"+pob+"/.luarocks/share/lua/5.1/?.lua;"+pob+"/.luarocks/share/lua/5.1/?/init.lua;;",
			"LUA_CPATH="+pob+"/.luarocks/lib/lua/5.1/?.so;;")
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("the tree oracle failed: %v\n%s", err, stderr.String())
		}
		got, err := os.ReadFile(out) //nolint:gosec // the test's own temp dir
		mustOK(t, err)
		want, err := os.ReadFile(filepath.Join("testdata", "tree_specs_"+scenario+"_pob.jsonl"))
		mustOK(t, err)
		if !bytes.Equal(got, want) {
			t.Errorf("testdata/tree_specs_%s_pob.jsonl is stale; PoB-PoE2 now writes:\n%s", scenario, got)
		}
	}
}
