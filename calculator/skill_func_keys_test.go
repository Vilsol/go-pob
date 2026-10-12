package calculator

import (
	"os"
	"sort"
	"testing"

	"github.com/Vilsol/go-pob/data/raw"
)

// TestSkillFuncKeysResolve is a REPORT-ONLY census: which entries in the skill-function registry are keyed on a granted-effect id that does not exist?
func TestSkillFuncKeysResolve(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true — this census needs the game data loaded")
	}
	requireGameData(t, "3.29")

	known := map[string]bool{}
	for id := range raw.PoBSkills {
		known[id] = true
	}
	for id := range raw.SkillBaseFlags {
		known[id] = true
	}
	if len(known) < 500 {
		t.Fatalf("REFUSING: only %d granted-effect ids loaded; the census cannot tell a dead key "+
			"from an unloaded one", len(known))
	}
	if !known["Cyclone"] {
		t.Fatal("REFUSING: control failed — `Cyclone` is not a known granted-effect id, so the id " +
			"set is wrong and every key would report as dead")
	}
	if known["ZzNotARealGrantedEffectId"] {
		t.Fatal("REFUSING: control failed — a synthetic id resolved, so the lookup matches anything")
	}

	var dead []string
	for id := range skillFuncs {
		if !known[id] {
			dead = append(dead, id)
		}
	}
	sort.Strings(dead)

	t.Logf("SKILL-FUNC KEYS: %d of %d registered ids match no granted effect", len(dead), len(skillFuncs))
	for _, id := range dead {
		t.Logf("    %s", id)
	}
	t.Log("A dead key may be a removed skill (harmless) or a RENAME (the function silently never " +
		"runs). Only reading the pinned PoB separates them.")
}
