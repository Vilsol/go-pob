package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
)

// TestPrecisionAuraCritReachesThePlayer is the end-to-end pin on the stat-key skew that skillStatKeyAliases exists for.
func TestPrecisionAuraCritReachesThePlayer(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}
	requireGameData(t, "3.29")

	const name = "15350-juggernaut-sunder.xml"

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	rec, ok := oracle[name]
	if !ok {
		t.Fatalf("REFUSING: %s is not in the oracle blob", name)
	}

	file, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), name))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	build, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}

	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
	if env == nil {
		t.Fatal("REFUSING: the build did not calculate")
	}

	for _, key := range []string{"PreEffectiveCritChance", "CritChance"} {
		want, ok := rec.Output[key]
		if !ok {
			t.Fatalf("REFUSING: the blob has no %s for %s", key, name)
		}
		got := env.Player.Output[key]
		if !oracleClose(got, want, oracleDumpTolerance) {
			t.Errorf("%s: got %v, PoB %v", key, got, want)
		}
	}
}

// TestSkillStatKeyAliasesAreAllLive keeps the alias table from rotting into a list of renames nobody needs any more.
func TestSkillStatKeyAliasesAreAllLive(t *testing.T) {
	if raw.LatestVersion != "3.20" {
		t.Skipf("SKIP: the alias population is measured against 3.20 data, process loaded %s",
			raw.LatestVersion)
	}

	live := map[string]int{}
	for _, ge := range poe.GrantedEffects {
		perSkill := raw.SkillStatMaps[ge.ID]
		if perSkill == nil {
			continue
		}
		for _, dataKey := range reachableStatKeys(ge) {
			pobKey, aliased := skillStatKeyAliases[dataKey]
			if !aliased {
				continue
			}
			if perSkill[dataKey] == nil && perSkill[pobKey] != nil {
				live[dataKey]++
			}
		}
	}

	for dataKey, pobKey := range skillStatKeyAliases {
		if live[dataKey] == 0 {
			t.Errorf("alias %q -> %q is DEAD: no granted effect has go-pob-data emitting %q "+
				"while its PoB statMap declares only %q. Either go-pob-data adopted the rename "+
				"(delete the alias) or the PoB pin moved (re-measure).", dataKey, pobKey, dataKey, pobKey)
		}
	}
	t.Logf("alias population: %v", live)
}

func reachableStatKeys(ge *poe.GrantedEffect) []string {
	c := raw.GetCalculatedGrantedEffect(ge)
	seen := map[string]bool{}
	for _, s := range c.GetCalculatedStats() {
		seen[s] = true
	}
	for k := range c.GetCalculatedConstantStats() {
		seen[k] = true
	}
	for _, qs := range ge.GetEffectQualityStats() {
		if qs == nil {
			continue
		}
		for _, s := range qs.GetStats() {
			seen[s.ID] = true
		}
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	return keys
}
