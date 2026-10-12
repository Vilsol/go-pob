package calculator

import (
	"os"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

// TestEveryKeystoneGrantsItsHaveCondition pins the SECOND mod PoB's generated keystone rows emit.
//
//	for _, name in ipairs(data.keystones) do
//			flag("Condition:Have"..firstToUpper(name):gsub(" %l", string.upper):gsub(" ", "")) }
//	end
func TestEveryKeystoneGrantsItsHaveCondition(t *testing.T) {
	if len(data.Keystones) == 0 {
		t.Fatalf("REFUSING: data.Keystones is empty, so this test would pass vacuously")
	}

	missing := make([]string, 0)
	for _, name := range data.Keystones {
		mods, ok := specialModList[strings.ToLower(name)]
		if !ok {
			t.Errorf("no specialModList entry for keystone %q at all", name)
			continue
		}
		list, ok := mods.([]mod.Mod)
		if !ok {
			t.Errorf("keystone %q maps to %T, not a plain mod list", name, mods)
			continue
		}
		want := "Condition:Have" + keystoneConditionSuffix(name)
		found := false
		for _, m := range list {
			if m.Name() == want {
				found = true
			}
		}
		if !found {
			missing = append(missing, name+" -> "+want)
		}
	}
	if len(missing) > 0 {
		t.Errorf("%d of %d keystones grant no Condition:Have flag; first few: %v",
			len(missing), len(data.Keystones), missing[:min(4, len(missing))])
	}
}

// TestGhostReaverConditionReachesTheBuild is the end-to-end half.
func TestGhostReaverConditionReachesTheBuild(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0161-assassin-viper-strike-of-the-mamba.xml")
	if !env.Player.ModDB.Flag(nil, "Condition:HaveGhostReaver") {
		t.Errorf("Condition:HaveGhostReaver is FALSE on a build that allocates Ghost Reaver")
	}

	ctrl := corpusEnvForLoadedData(t, "0004-trickster-lightning-tendrils-of-eccentricity.xml")
	if ctrl.Player.ModDB.Flag(nil, "Condition:HaveGhostReaver") {
		t.Errorf("CONTROL: Condition:HaveGhostReaver is set on a build without the keystone")
	}
}
