package calculator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data"
)

// TestCombineStatOrLeavesAbsentStatsAbsent pins CalcOffence.lua:2005's `output[stat] = output.MainHand[stat] or output.OffHand[stat]`.
func TestCombineStatOrLeavesAbsentStatsAbsent(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	corpusDir, _ := corpusNamesForLoadedData(t)
	entries, err := os.ReadDir(corpusDir)
	if err != nil {
		t.Fatalf("%v", err)
	}

	sources, presentZero, absent, presentNonZero := 0, 0, 0, 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".xml") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			continue
		}
		if !strings.Contains(string(raw), "Cast On Critical Strike") {
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
		ms := env.Player.MainSkill
		if ms.TriggeredBy == nil || ms.TriggeredBy.GrantedEffect == nil ||
			ms.TriggeredBy.GrantedEffect.ID() != "SupportCastOnCritTriggered" {
			continue
		}
		for _, sk := range env.Player.ActiveSkillList {
			if !sk.SkillTypes[data.SkillTypeAttack] || sk == ms || sk.SocketGroupIndex != ms.SocketGroupIndex {
				continue
			}
			cached := buildActiveSkillCache(env, sk)
			if cached == nil {
				continue
			}
			sources++
			switch {
			case !cached.hitSpeedSet:
				absent++
			case cached.HitSpeed == 0:
				presentZero++
			default:
				presentNonZero++
			}
		}
	}

	if sources < 50 {
		t.Fatalf("REFUSING: only %d Cast-on-Crit source skills reached; the population has moved", sources)
	}
	if presentZero != 0 {
		t.Errorf("HitSpeed is present-but-zero on %d of %d cached sources; combineStat is manufacturing a key Lua leaves nil (absent %d, present-nonzero %d)",
			presentZero, sources, absent, presentNonZero)
	}
	t.Logf("cached CoC sources %d: absent %d, present-nonzero %d, present-but-zero %d", sources, absent, presentNonZero, presentZero)
}
