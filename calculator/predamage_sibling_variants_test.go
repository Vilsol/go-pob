package calculator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var preDamageSiblingArms = []struct {
	skillID   string
	parts     []int
	wantInert bool
	pob       string
}{
	{"BarrageAltX", []int{2}, true, "act_dex.lua:1114-1118"},
	{"VaalLightningArrow", []int{2}, true, "act_dex.lua:10587-10591"},
	{"IceSpearAltX", []int{3, 4}, true, "act_int.lua:9915-9918"},
	{"InfernalBlowAltX", []int{2, 3}, false, "act_str.lua:7178-7186"},
}

func TestTransfiguredPreDamageSiblingsRunTheirArm(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	dir, names := corpusNamesForLoadedData(t)

	for _, arm := range preDamageSiblingArms {
		t.Run(arm.skillID, func(t *testing.T) {
			wantPart := map[int]bool{}
			for _, p := range arm.parts {
				wantPart[p] = true
			}

			var carriers, reached, withMods int
			for _, name := range names {
				raw, err := os.ReadFile(filepath.Join(dir, name))
				if err != nil {
					t.Fatalf("REFUSING: %v", err)
				}
				if !strings.Contains(string(raw), `skillId="`+arm.skillID+`"`) {
					continue
				}
				rec, ok := oracle[name]
				if !ok || rec.TreeSources < oracleMinTreeSources {
					continue
				}
				carriers++
				env := corpusEnvForLoadedData(t, name)
				ms := env.Player.MainSkill
				if ms == nil || ms.ActiveEffect == nil || ms.ActiveEffect.GrantedEffect == nil {
					continue
				}
				if ms.ActiveEffect.GrantedEffect.ID() != arm.skillID || !wantPart[ms.SkillPart] {
					continue
				}
				reached++

				for _, m := range ms.SkillModList.Mods() {
					if m.Name() == "Damage" && string(m.GetSource()) == "Skill:"+arm.skillID {
						withMods++
						break
					}
				}
			}

			if carriers == 0 {
				t.Fatalf("REFUSING: no graded corpus build sockets %s at all, so this subtest "+
					"measures nothing", arm.skillID)
			}
			t.Logf("%s: %d graded carriers, %d reach part %v", arm.skillID, carriers, reached, arm.parts)

			if arm.wantInert {
				if reached > 0 {
					t.Errorf("%s is recorded INERT but %d graded build(s) now reach part %v — "+
						"the registration decides numbers again and needs a real fixture (%s)",
						arm.skillID, reached, arm.parts, arm.pob)
				}
				return
			}
			if reached == 0 {
				t.Fatalf("REFUSING: %s is recorded reachable but no graded build runs it as main "+
					"skill on part %v; deleting the registration would still pass", arm.skillID, arm.parts)
			}
			if withMods != reached {
				t.Errorf("%d of %d builds reaching %s's arm carry a Damage mod sourced "+
					"Skill:%s; %s emits one on every part-2 and part-3 build",
					withMods, reached, arm.skillID, arm.skillID, arm.pob)
			}
		})
	}
}
