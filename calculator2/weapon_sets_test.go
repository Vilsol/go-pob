package calculator2

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/pob2"
)

// weaponSetsFixtures are builds whose groups have a weapon set they cannot be used in, or must be in both.
var weaponSetsFixtures = []string{"crossbow_plain", "mana_regen_to_rage", "radius_jewel"}

// The Set 1 and Set 2 boxes of every group are PoB-PoE2's (testdata/weapon_sets_<build>_pob.json, from
// testdata/weapon_sets_oracle.lua).
func TestSocketGroupWeaponSetsMatchPoB(t *testing.T) {
	tree := loadTestTree(t)
	for _, name := range weaponSetsFixtures {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("testdata", "weapon_sets_"+name+"_pob.json"))
			if err != nil {
				t.Fatal(err)
			}
			var oracle []struct {
				Group int `json:"group"`
				Sets  []struct {
					Valid   bool `json:"valid"`
					Enabled bool `json:"enabled"`
				} `json:"sets"`
				ForcedBoth bool `json:"forcedBoth"`
			}
			if err := json.Unmarshal(raw, &oracle); err != nil {
				t.Fatal(err)
			}
			xml, err := os.ReadFile(filepath.Join("testdata", name+".xml"))
			if err != nil {
				t.Fatal(err)
			}
			build, err := pob2.ParseBuild(xml)
			if err != nil {
				t.Fatal(err)
			}
			env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
			if err != nil {
				t.Fatal(err)
			}
			if len(env.SocketGroupList) != len(oracle) {
				t.Fatalf("%d groups, PoB-PoE2 %d", len(env.SocketGroupList), len(oracle))
			}
			for _, want := range oracle {
				valid, enabled, forcedBoth := SocketGroupWeaponSets(env, want.Group)
				for set := range 2 {
					if valid[set] != want.Sets[set].Valid || enabled[set] != want.Sets[set].Enabled {
						t.Errorf("group %d set %d: valid %v enabled %v, PoB-PoE2 %v %v", want.Group, set+1, valid[set], enabled[set], want.Sets[set].Valid, want.Sets[set].Enabled)
					}
				}
				if forcedBoth != want.ForcedBoth {
					t.Errorf("group %d: forcedBoth %v, PoB-PoE2 %v", want.Group, forcedBoth, want.ForcedBoth)
				}
			}
		})
	}
}

// The committed oracle is what testdata/weapon_sets_oracle.lua writes from the pinned PoB-PoE2; CORPUS=true runs it.
func TestWeaponSetsOracleIsCurrent(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true runs PoB-PoE2")
	}
	pob, luajit := pobPoE2Probe(t)
	probe, err := filepath.Abs(filepath.Join("testdata", "weapon_sets_oracle.lua"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range weaponSetsFixtures {
		fixture, err := filepath.Abs(filepath.Join("testdata", name+".xml"))
		if err != nil {
			t.Fatal(err)
		}
		out := filepath.Join(t.TempDir(), "weapon_sets.json")
		cmd := exec.CommandContext(context.Background(), "setarch", "-R", luajit, probe, out, fixture)
		cmd.Dir = filepath.Join(pob, "src")
		cmd.Env = append(os.Environ(),
			"LUA_PATH="+pob+"/runtime/lua/?.lua;"+pob+"/runtime/lua/?/init.lua;"+pob+"/.luarocks/share/lua/5.1/?.lua;"+pob+"/.luarocks/share/lua/5.1/?/init.lua;;",
			"LUA_CPATH="+pob+"/.luarocks/lib/lua/5.1/?.so;;")
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("the weapon sets oracle failed: %v\n%s", err, stderr.String())
		}
		got, err := os.ReadFile(out) //nolint:gosec // the test's own temp dir
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(filepath.Join("testdata", "weapon_sets_"+name+"_pob.json"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("testdata/weapon_sets_%s_pob.json is stale; PoB-PoE2 now writes:\n%s", name, got)
		}
	}
}
