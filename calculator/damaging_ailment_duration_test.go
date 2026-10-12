package calculator

import (
	"os"
	"testing"
)

// TestDamagingAilmentDurationIsTheNamePoBUses pins the mod NAME the three ailment-duration sites read, which this tree had invented.
func TestDamagingAilmentDurationIsTheNamePoBUses(t *testing.T) {
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

	keys := []string{"IgniteDuration", "BleedDuration", "PoisonDuration"}
	var decisive, seen, wrong, invented int

	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		ms := env.Player.MainSkill
		if ms == nil || ms.SkillModList == nil {
			continue
		}
		if ms.SkillModList.More(ms.SkillCfg, "SkillAndDamagingAilmentDuration") != 1 {
			invented++
		}
		if ms.SkillModList.More(ms.SkillCfg, "DamagingAilmentDuration") == 1 {
			continue
		}
		decisive++

		for _, key := range keys {
			want, has := rec.Output[key]
			if !has || want == 0 {
				continue
			}
			seen++
			if !oracleClose(env.Player.Output[key], want, oracleDumpTolerance) {
				wrong++
			}
		}
		if decisive >= 60 {
			break
		}
	}

	if invented > 0 {
		t.Errorf("`SkillAndDamagingAilmentDuration` is PRODUCED on %d build(s). It appears in 0 "+
			"files of the pinned PoB, so a producer means this tree grew one and dropping the name "+
			"from the skill-duration sites is no longer inert", invented)
	}
	if decisive == 0 {
		t.Fatal("REFUSING: no graded build carries a live DamagingAilmentDuration MORE on its main " +
			"skill (Swift Affliction is the producer), so this test measures nothing")
	}
	if seen == 0 {
		t.Fatal("REFUSING: the blob publishes no nonzero ailment duration on those builds")
	}
	if wrong > 0 {
		t.Logf("ailment durations still disagree with PoB on %d of %d pairs across %d builds "+
			"carrying a DamagingAilmentDuration multiplier (a separate residual)", wrong, seen, decisive)
	}
}
