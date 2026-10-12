package calculator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var dumpTreeSourceRegex = regexp.MustCompile(`^Tree:(\d+) \((.*)\)$`)

// TestAbyssZorathConquestsMatchPoBDumps grades applyAbyssConquests against the pinned PoB's own per-build dumps.
func TestAbyssZorathConquestsMatchPoBDumps(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	dumps := dumpsDirForLoadedData(t)
	corpus := corpusDirForLoadedData(t)

	entries, err := os.ReadDir(corpus)
	if err != nil {
		t.Fatal(err)
	}

	carriers, replaced, forbidden := 0, 0, 0
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".xml") {
			continue
		}
		xml, err := os.ReadFile(filepath.Join(corpus, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(xml), "Assembled Eye Jewel") {
			continue
		}
		carriers++

		raw, err := os.ReadFile(filepath.Join(dumps, strings.TrimSuffix(e.Name(), ".xml")+".json"))
		if err != nil {
			t.Errorf("%s: no PoB dump: %v", e.Name(), err)
			continue
		}
		var dump struct {
			ModDB map[string][]struct {
				Source string `json:"source"`
			} `json:"modDB"`
		}
		if err := json.Unmarshal(raw, &dump); err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		pobDN := make(map[string]string)
		for _, mods := range dump.ModDB {
			for _, m := range mods {
				if match := dumpTreeSourceRegex.FindStringSubmatch(m.Source); match != nil {
					pobDN[match[1]] = match[2]
				}
			}
		}

		env := corpusEnvForLoadedData(t, e.Name())
		rewrites := nodeRewrites(env)
		for id, dn := range pobDN {
			node, allocated := env.AllocatedNodes[id]
			if !allocated || node.Name == nil {
				continue
			}
			rewrite := rewrites[id]
			if env.GrantedAscendancyNodes[id] {
				forbidden++
				if rewrite.stamped {
					t.Errorf("%s: node %s (%s) is granted by Forbidden Flame/Flesh, which reads the TREE "+
						"node (CalcSetup.lua:1308), but carries a conquest", e.Name(), id, *node.Name)
				}
				continue
			}
			ours := *node.Name
			if r := rewrite.effectiveReplacement(); r != nil {
				ours = r.DN
			}
			if ours != *node.Name {
				replaced++
			}
			if ours != dn {
				t.Errorf("%s: node %s (%s) is %q on our side, PoB's is %q", e.Name(), id, *node.Name, ours, dn)
			}
		}
	}

	t.Logf("%d carriers, %d replaced nodes agree, %d Forbidden-granted nodes checked", carriers, replaced, forbidden)
	if carriers == 0 || replaced == 0 || forbidden == 0 {
		t.Fatalf("REFUSING: %d carriers / %d replaced / %d forbidden — a zero means this asserted nothing",
			carriers, replaced, forbidden)
	}
}
