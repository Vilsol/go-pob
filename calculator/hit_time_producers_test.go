package calculator

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestHitTimeIsWrittenWhereverPoBWritesIt guards the STARVED-PRODUCER class behind output.HitTime, rather than any one skill's formula.
func TestHitTimeIsWrittenWhereverPoBWritesIt(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)
	entries, err := os.ReadDir(corpusDir)
	if err != nil {
		t.Fatalf("%v", err)
	}

	expected, starved := 0, 0
	bySkill := map[string]int{}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".xml") {
			continue
		}
		rec, ok := oracle[name]
		if !ok || rec.Output["HitTime"] == 0 {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			continue
		}
		build, err := builds.ParseBuild(raw)
		if err != nil {
			continue
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		if env == nil || env.Player == nil || env.Player.MainSkill == nil {
			continue
		}
		expected++
		if env.Player.Output["HitTime"] != 0 {
			continue
		}
		starved++
		id := "<none>"
		if ae := env.Player.MainSkill.ActiveEffect; ae != nil && ae.GrantedEffect != nil {
			id = ae.GrantedEffect.ID()
		}
		bySkill[id]++
	}

	if expected < 900 {
		t.Fatalf("REFUSING: only %d builds publish HitTime in the blob; the population has moved", expected)
	}

	const starvedCeiling = 56
	if starved > starvedCeiling {
		type kv struct {
			k string
			v int
		}
		l := make([]kv, 0, len(bySkill))
		for k, v := range bySkill {
			l = append(l, kv{k, v})
		}
		sort.Slice(l, func(i, j int) bool { return l[i].v > l[j].v })
		top := l
		if len(top) > 8 {
			top = top[:8]
		}
		t.Errorf("HitTime starved on %d of %d builds, ceiling %d; worst: %v", starved, expected, starvedCeiling, top)
	}
	t.Logf("HitTime: %d builds expect it, %d starved (%d distinct skills)", expected, starved, len(bySkill))
}
