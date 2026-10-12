package calculator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestHaveDamageableMinionIsWrittenPerActiveSkill pins the SCOPE of CalcPerform.lua:1426-1428.
func TestHaveDamageableMinionIsWrittenPerActiveSkill(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	corpusDir, _ := corpusNamesForLoadedData(t)
	dumps := dumpsDirForLoadedData(t)

	const name = "0010-elementalist-storm-burst-2.xml"

	raw, err := os.ReadFile(filepath.Join(dumps, strings.TrimSuffix(name, ".xml")+".json"))
	if err != nil {
		t.Fatalf("%v", err)
	}
	var dump struct {
		ModDB map[string][]json.RawMessage `json:"modDB"`
	}
	if err := json.Unmarshal(raw, &dump); err != nil {
		t.Fatalf("%v", err)
	}
	want := len(dump.ModDB["Condition:HaveDamageableMinion"])
	if want != 5 {
		t.Fatalf("REFUSING: PoB's dump for %s carries %d Condition:HaveDamageableMinion mods, "+
			"not the 5 this fixture was built on — the build or the dump set changed", name, want)
	}

	file, err := os.ReadFile(filepath.Join(corpusDir, name))
	if err != nil {
		t.Fatalf("%v", err)
	}
	build, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatalf("%v", err)
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())

	if got := len(env.ModDB.Mods["Condition:HaveDamageableMinion"]); got != want {
		t.Errorf("%s: %d Condition:HaveDamageableMinion mods, want %d (PoB's dump); 4 means the "+
			"write is still inside CalcPerform.lua:1403's minionList guard", name, got, want)
	}
}
