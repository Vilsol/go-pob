package calculator

import (
	"math"
	"os"
	"strings"
	"testing"
)

func assertCorpusStatsMatchOracle(t *testing.T, builds []string, stats ...string) {
	t.Helper()
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	for _, build := range builds {
		t.Run(build, func(t *testing.T) {
			rec, ok := oracle[build]
			if !ok {
				t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
			}
			env := corpusEnvForLoadedData(t, build)
			for _, stat := range stats {
				var want, got float64
				for _, pair := range oracleTablePairs(rec, env) {
					if k, ok := strings.CutPrefix(stat, pair.prefix); ok && !strings.Contains(k, ".") {
						want, got = pair.exp[k], pair.got[k]
					}
				}
				if want == 0 {
					t.Fatalf("REFUSING: PoB reports %s 0 for %s", stat, build)
				}
				if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
					t.Errorf("%s = %v, PoB reports %v (ratio %.4f)", stat, got, want, got/want)
				}
			}
		})
	}
}

// TestTornadoShotSecondaryProjectilesReachTheOracle: ModParser.lua:4533-4534.
func TestTornadoShotSecondaryProjectilesReachTheOracle(t *testing.T) {
	assertCorpusStatsMatchOracle(t, []string{
		"15768-pathfinder-tornado-shot.xml",
		"15856-pathfinder-tornado-shot.xml",
	}, "TotalDPS", "CombinedDPS")
}

// TestExplosiveTrapOfShrapnelBorrowsPartsReachTheOracle: act_dex.lua:7321-7322.
func TestExplosiveTrapOfShrapnelBorrowsPartsReachTheOracle(t *testing.T) {
	assertCorpusStatsMatchOracle(t, []string{
		"15841-saboteur-explosive-trap-of-shrapnel.xml",
	}, "OverlapChance", "TotalDPS")
}

// TestSoulLinkFromPartyMemberReachesTheOracle: CalcDefence.lua:2517-2526.
func TestSoulLinkFromPartyMemberReachesTheOracle(t *testing.T) {
	assertCorpusStatsMatchOracle(t, []string{
		"15851-inquisitor-petrified-blood.xml",
	}, "SoulLinkMitigation", "AlliedEnergyShield", "TotalEHP")
}

// TestMinionsTakeReducedDamageReachesTheOracle: ModParser.lua:4679.
func TestMinionsTakeReducedDamageReachesTheOracle(t *testing.T) {
	assertCorpusStatsMatchOracle(t, []string{
		"15666-gladiator-raise-zombie.xml",
	}, "Minion.TotalEHP")
}

// TestGrantedPassiveFallsBackToAscendancyMapReachesTheOracle: CalcSetup.lua:1288-1290.
func TestGrantedPassiveFallsBackToAscendancyMapReachesTheOracle(t *testing.T) {
	assertCorpusStatsMatchOracle(t, []string{
		"15769-pathfinder-fireball.xml",
	}, "IgniteChanceOnHit", "IgniteDPS")
}

// TestPartyLinkPercentStatReadsPartyMemberReachesTheOracle: ModTools.lua:50-76, PartyTab.lua:878-885, CalcPerform.lua:1276.
func TestPartyLinkPercentStatReadsPartyMemberReachesTheOracle(t *testing.T) {
	assertCorpusStatsMatchOracle(t, []string{
		"15748-assassin-firestorm-of-meteors.xml",
	}, "FireMinBase", "FireMaxBase", "TotalDPS")
}

// TestHallowingFlameExtraAuraReachesTheOracle: CalcPerform.lua:3345-3349.
func TestHallowingFlameExtraAuraReachesTheOracle(t *testing.T) {
	assertCorpusStatsMatchOracle(t, []string{
		"15902-guardian-holy-strike.xml",
	}, "IgniteDPS", "TotalDPS")
}
