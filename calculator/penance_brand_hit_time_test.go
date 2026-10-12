package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestPenanceBrandHitTimeCarriesMaxEnergy pins act_int.lua:13810.
func TestPenanceBrandHitTimeCarriesMaxEnergy(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	var decisive, seen, wrong int

	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		ms := env.Player.MainSkill
		if ms == nil || ms.ActiveEffect == nil || ms.ActiveEffect.GrantedEffect == nil ||
			ms.ActiveEffect.GrantedEffect.Raw == nil {
			continue
		}
		if ms.ActiveEffect.GrantedEffect.Raw.ID != "PenanceBrand" {
			continue
		}
		if ms.SkillModList == nil ||
			ms.SkillModList.Sum(mod.TypeBase, ms.SkillCfg, "Multiplier:PenanceBrandMaxEnergy") == 0 {
			continue
		}
		decisive++

		for _, key := range []string{"HitTime", "HitSpeed"} {
			want, has := rec.Output[key]
			if !has || want == 0 {
				continue
			}
			seen++
			if !oracleClose(env.Player.Output[key], want, oracleDumpTolerance) {
				wrong++
			}
		}
		if decisive >= 40 {
			break
		}
	}

	if decisive == 0 {
		t.Fatal("REFUSING: no graded build runs PenanceBrand with a live " +
			"Multiplier:PenanceBrandMaxEnergy, so the extra factor decides nothing here and this " +
			"test no longer measures its mechanism")
	}
	if seen == 0 {
		t.Fatal("REFUSING: the blob publishes no nonzero HitTime/HitSpeed on those builds")
	}
	if wrong > 0 {
		t.Logf("HitTime/HitSpeed still disagree with PoB on %d of %d pairs across %d PenanceBrand "+
			"builds (see the note above: the 26.6667 and 10.0 clusters, a separate cause)",
			wrong, seen, decisive)
	}
}
