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
	"strings"
	"testing"
)

// treeJewelsRow is one row of testdata/tree_jewels_pob.jsonl, which testdata/tree_jewels_oracle.lua writes from the
// pinned PoB-PoE2.
type treeJewelsRow struct {
	Step    string         `json:"step"`
	Sockets map[string]int `json:"sockets"`
	Output  map[string]any `json:"output"`
}

// PoE2 jewels (Diamond, Sapphire, Ruby, Emerald) in tree sockets, and the same socket edits through the Items tab:
// every numeric main output and what each socket holds are PoB-PoE2's after each step.
func TestTreeJewelsMatchPoB(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "tree_jewels_pob.jsonl"))
	mustOK(t, err)
	defer f.Close()
	var rows []treeJewelsRow
	sc := bufio.NewScanner(f)
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		var r treeJewelsRow
		mustOK(t, json.Unmarshal(sc.Bytes(), &r))
		rows = append(rows, r)
	}
	mustOK(t, sc.Err())
	if len(rows) != 4 {
		t.Fatalf("%d oracle rows, want 4", len(rows))
	}
	calc, out := openFixture(t, "sorceress_tree_jewels.xml")
	edits := map[string]func() error{
		"empty socket 21984":         func() error { return calc.SetSlotItem("Jewel 21984", 0) },
		"Rune Spark in socket 21984": func() error { return calc.SetSlotItem("Jewel 21984", 7) },
		"empty socket 7960":          func() error { return calc.SetSlotItem("Jewel 7960", 0) },
	}
	for _, want := range rows {
		if want.Step != "load" {
			mustOK(t, edits[want.Step]())
			out, err = calc.Calculate()
			mustOK(t, err)
		}
		page, err := calc.ItemsPage()
		mustOK(t, err)
		got := map[string]int{}
		for _, s := range page.Slots {
			if s.NodeID != 0 && s.ItemID != 0 {
				got[s.Name] = s.ItemID
			}
		}
		if len(got) != len(want.Sockets) {
			t.Errorf("%s: sockets %v, PoB-PoE2 %v", want.Step, got, want.Sockets)
		}
		for name, id := range want.Sockets {
			if got[name] != id {
				t.Errorf("%s: sockets %v, PoB-PoE2 %v", want.Step, got, want.Sockets)
				break
			}
		}
		for k, v := range want.Output {
			w, ok := v.(float64)
			if !ok {
				continue
			}
			g := out.Output[k]
			if math.Abs(g-w) > 1e-9*math.Max(1, math.Abs(w)) {
				t.Errorf("%s: %s = %v, PoB-PoE2 %v", want.Step, k, g, w)
			}
		}
	}
}

// The committed oracle rows are what testdata/tree_jewels_oracle.lua writes from the pinned PoB-PoE2; CORPUS=true
// runs it, as it starts PoB-PoE2.
func TestTreeJewelsOracleIsCurrent(t *testing.T) {
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
	probe, err := filepath.Abs(filepath.Join("testdata", "tree_jewels_oracle.lua"))
	mustOK(t, err)
	fixture, err := filepath.Abs(filepath.Join("testdata", "sorceress_tree_jewels.xml"))
	mustOK(t, err)
	out := filepath.Join(t.TempDir(), "jewels.jsonl")
	cmd := exec.CommandContext(context.Background(), "setarch", "-R", luajit, probe, out, fixture)
	cmd.Dir = filepath.Join(pob, "src")
	cmd.Env = append(os.Environ(),
		"LUA_PATH="+pob+"/runtime/lua/?.lua;"+pob+"/runtime/lua/?/init.lua;"+pob+"/.luarocks/share/lua/5.1/?.lua;"+pob+"/.luarocks/share/lua/5.1/?/init.lua;;",
		"LUA_CPATH="+pob+"/.luarocks/lib/lua/5.1/?.so;;")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("the jewel oracle failed: %v\n%s", err, stderr.String())
	}
	got, err := os.ReadFile(out) //nolint:gosec // the test's own temp dir
	mustOK(t, err)
	want, err := os.ReadFile(filepath.Join("testdata", "tree_jewels_pob.jsonl"))
	mustOK(t, err)
	if !bytes.Equal(got, want) {
		t.Errorf("testdata/tree_jewels_pob.jsonl is stale; PoB-PoE2 now writes:\n%s", got)
	}
}
