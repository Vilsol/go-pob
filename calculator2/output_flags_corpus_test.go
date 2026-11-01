package calculator2

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/pob2"
)

// TestOutputFlagsVsPoBPoE2Corpus runs the first FLAG_BUILDS (default 30) PoE2 corpus builds through
// PoB-PoE2 and Go and compares the boolean outputs: a flag is set when PoB's output holds true. Keys
// Go never writes on any of the builds are reported, not failed. The string outputs must be equal.
func TestOutputFlagsVsPoBPoE2Corpus(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("SKIP: reads the local PoE2 corpus; set CORPUS=true")
	}
	dir := filepath.Join("..", "testdata", "corpus-poe2-0.5")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("SKIP: no PoE2 corpus at %s: %v", dir, err)
	}
	n := 30
	if s := os.Getenv("FLAG_BUILDS"); s != "" {
		if n, err = strconv.Atoi(s); err != nil {
			t.Fatalf("FLAG_BUILDS=%q: %v", s, err)
		}
	}
	var xmls []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".xml") && len(xmls) < n {
			p, err := filepath.Abs(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			xmls = append(xmls, p)
		}
	}
	if len(xmls) == 0 {
		t.Skipf("SKIP: no XMLs in %s", dir)
	}
	pob, pobStrings := runOutputFlagsOracle(t, xmls)
	if len(pob) != len(xmls) {
		t.Fatalf("the oracle printed %d builds of %d", len(pob), len(xmls))
	}

	type tally struct{ pobTrue, written, ok, miss, extra int }
	perKey := map[string]*tally{}
	stringsSeen := 0
	tree := loadTestTree(t)
	for _, x := range xmls {
		name := filepath.Base(x)
		raw, err := os.ReadFile(x) //nolint:gosec // the corpus path
		if err != nil {
			t.Fatal(err)
		}
		build, err := pob2.ParseBuild(raw)
		if err != nil {
			t.Fatal(err)
		}
		env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for key, want := range pobStrings[name] {
			stringsSeen++
			if got, ok := env.Player.OutputStrings[key]; !ok || got != want {
				t.Errorf("%s: %s is %q in PoB, ours %q (written %v)", name, key, want, got, ok)
			}
		}
		for key, got := range env.Player.OutputStrings {
			if _, ok := pobStrings[name][key]; !ok {
				t.Errorf("%s: %s is %q in ours, not a string in PoB", name, key, got)
			}
		}
		for key := range keysOf(pob[name], env.Player.OutputFlags) {
			want := pob[name][key]
			got, written := env.Player.OutputFlags[key]
			tl := perKey[key]
			if tl == nil {
				tl = &tally{}
				perKey[key] = tl
			}
			if written {
				tl.written++
			}
			switch {
			case want && got:
				tl.pobTrue++
				tl.ok++
			case want:
				tl.pobTrue++
				tl.miss++
			case got:
				tl.extra++
			}
		}
	}
	keys := make([]string, 0, len(perKey))
	for k := range perKey {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var written, unwritten []string
	for _, k := range keys {
		tl := perKey[k]
		if tl.written == 0 {
			unwritten = append(unwritten, k+" ("+strconv.Itoa(tl.pobTrue)+")")
			continue
		}
		written = append(written, k)
		t.Logf("  %-44s PoB true %4d  ok %4d  miss %4d  extra %4d", k, tl.pobTrue, tl.ok, tl.miss, tl.extra)
		if tl.miss > 0 || tl.extra > 0 {
			t.Errorf("%s: %d builds PoB sets and we do not, %d we set and PoB does not", k, tl.miss, tl.extra)
		}
	}
	t.Logf("%d flags written and graded over %d builds; not written (PoB true count): %s", len(written), len(xmls), strings.Join(unwritten, ", "))
	t.Logf("%d string outputs graded", stringsSeen)
	if len(written) == 0 {
		t.Fatal("no flag was graded: Actor.OutputFlags is never written")
	}
}

func keysOf(a, b map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k := range a {
		out[k] = true
	}
	for k := range b {
		out[k] = true
	}
	return out
}

// runOutputFlagsOracle returns, per build file name, the boolean outputs PoB-PoE2 sets true and its string outputs.
func runOutputFlagsOracle(t *testing.T, xmls []string) (map[string]map[string]bool, map[string]map[string]string) {
	t.Helper()
	pob, luajit := pobPoE2Probe(t)
	probe, err := filepath.Abs(filepath.Join("testdata", "output_flags_oracle.lua"))
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "flags.tsv")
	cmd := exec.CommandContext(context.Background(), "setarch", append([]string{"-R", luajit, probe, out}, xmls...)...) //nolint:gosec // the pinned luajit, the probe and corpus paths
	cmd.Dir = filepath.Join(pob, "src")
	cmd.Env = append(os.Environ(),
		"LUA_PATH="+pob+"/runtime/lua/?.lua;"+pob+"/runtime/lua/?/init.lua;"+pob+"/.luarocks/share/lua/5.1/?.lua;"+pob+"/.luarocks/share/lua/5.1/?/init.lua;;",
		"LUA_CPATH="+pob+"/.luarocks/lib/lua/5.1/?.so;;")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("the output flags oracle failed: %v\n%s", err, stderr.String())
	}
	b, err := os.ReadFile(out) //nolint:gosec // the test's own temp dir
	if err != nil {
		t.Fatal(err)
	}
	rows := map[string]map[string]bool{}
	stringRows := map[string]map[string]string{}
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		f := strings.Split(sc.Text(), "\t")
		if len(f) != 4 {
			t.Fatalf("bad oracle row %q", sc.Text())
		}
		if rows[f[0]] == nil {
			rows[f[0]] = map[string]bool{}
		}
		if f[2] == "boolean" && f[3] == "true" {
			rows[f[0]][f[1]] = true
		}
		if f[2] == "string" {
			if stringRows[f[0]] == nil {
				stringRows[f[0]] = map[string]string{}
			}
			stringRows[f[0]][f[1]] = f[3]
		}
	}
	return rows, stringRows
}
