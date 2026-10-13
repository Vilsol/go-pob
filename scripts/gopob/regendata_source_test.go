package main

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var rawGameDataStillFromPoB = []string{
	"tree-tables",
}

var gameDataDumperRe = regexp.MustCompile(`(?m)^\t"([a-z0-9-]+)": +dump[A-Za-z]+,$`)

// TestRawGameDataIsNotReadFromPoB: a table classified as raw game data comes from go-pob-data through scripts/gamedatadump.
func TestRawGameDataIsNotReadFromPoB(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "gamedatadump", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	dumpers := map[string]bool{}
	for _, m := range gameDataDumperRe.FindAllStringSubmatch(string(b), -1) {
		dumpers[m[1]] = true
	}
	if len(dumpers) == 0 {
		t.Fatal("no dumper parsed from scripts/gamedatadump/main.go; the scan has stopped matching")
	}

	fromPoB := 0
	for _, g := range rdGenerators {
		switch g.source {
		case rdGameData:
			if g.lua != "" || g.stdinGz != "" || g.twoTree {
				t.Errorf("%s is raw game data but reads PoB (lua %q, stdinGz %q, twoTree %v)", g.name, g.lua, g.stdinGz, g.twoTree)
			}
			if !dumpers[g.dump] {
				t.Errorf("%s is raw game data but its dump %q is not a scripts/gamedatadump table", g.name, g.dump)
			}
		case rdGameDataOverlay:
			if g.lua == "" || !dumpers[g.dump] {
				t.Errorf("%s overlays go-pob-data on a lua dumper but has lua %q, dump %q", g.name, g.lua, g.dump)
			}
		case rdGameDataFromPoB:
			fromPoB++
			if !slices.Contains(rawGameDataStillFromPoB, g.name) {
				t.Errorf("%s is raw game data read from PoB: derive it from go-pob-data instead", g.name)
			}
		case rdPoBOracle:
			if !strings.HasPrefix(g.out, "testdata/") {
				t.Errorf("%s is a PoB oracle but ships as %s", g.name, g.out)
			}
		case rdPoBAuthored:
			if g.dump != "" {
				t.Errorf("%s is PoB-authored but reads go-pob-data (dump %q): classify it rdGameData", g.name, g.dump)
			}
		default:
			t.Errorf("%s has no source: classify it", g.name)
		}
	}
	for _, name := range rawGameDataStillFromPoB {
		if !slices.ContainsFunc(rdGenerators, func(g rdGenerator) bool { return g.name == name && g.source == rdGameDataFromPoB }) {
			t.Errorf("%s no longer reads PoB: drop it from rawGameDataStillFromPoB", name)
		}
	}
	if fromPoB != len(rawGameDataStillFromPoB) {
		t.Errorf("%d tables read raw game data from PoB, rawGameDataStillFromPoB lists %d", fromPoB, len(rawGameDataStillFromPoB))
	}
}
