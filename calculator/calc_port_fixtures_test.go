package calculator

import (
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// Each fixture in testdata/calc-port-fixtures was run through the pinned PoB; the wanted values are its output.
func portFixtureEnv(t *testing.T, name string, mode OutputMode) *Environment {
	t.Helper()
	requireGameData(t, "3.30")
	raw, err := os.ReadFile(filepath.Join("testdata", "calc-port-fixtures", name+".xml"))
	if err != nil {
		t.Fatal(err)
	}
	build, err := builds.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	return BuildOutputWithCache(&Calculator{PoB: build}, mode, NewEnvironmentCache())
}

func wantStat(t *testing.T, fixture string, table map[string]float64, stat string, want float64) {
	t.Helper()
	got, ok := table[stat]
	if !ok {
		t.Errorf("%s: %s absent, PoB %v", fixture, stat, want)
		return
	}
	if math.Abs(got-want) > 1e-6*math.Max(1, math.Abs(want)) {
		t.Errorf("%s: %s = %v, PoB %v", fixture, stat, got, want)
	}
}

func wantAbsent(t *testing.T, fixture string, table map[string]float64, stat string) {
	t.Helper()
	if got, ok := table[stat]; ok {
		t.Errorf("%s: %s = %v, PoB leaves it unset", fixture, stat, got)
	}
}

// TestDaggerDetonationAddsDaggerDamageToSpells pins CalcOffence.lua:932-942.
func TestDaggerDetonationAddsDaggerDamageToSpells(t *testing.T) {
	for fixture, want := range map[string][3]float64{
		"dagger-detonation-dual":   {85.5, 138.75, 117.73125},
		"dagger-detonation-single": {84, 144.5, 119.9625},
	} {
		env := portFixtureEnv(t, fixture, OutputModeCalcs)
		wantStat(t, fixture, env.Player.Output, "PhysicalMin", want[0])
		wantStat(t, fixture, env.Player.Output, "PhysicalMax", want[1])
		wantStat(t, fixture, env.Player.Output, "AverageHit", want[2])
	}
}

// TestMinionCastSpeedAppliesToPlayer pins CalcOffence.lua:634-643.
func TestMinionCastSpeedAppliesToPlayer(t *testing.T) {
	const fixture = "minion-cast-speed-to-player"
	env := portFixtureEnv(t, fixture, OutputModeMain)
	wantStat(t, fixture, env.Player.Output, "Speed", 1.7333333333333)
}

// TestTrapCooldownWithoutBaseCooldown pins CalcOffence.lua:1296-1304.
func TestTrapCooldownWithoutBaseCooldown(t *testing.T) {
	const fixture = "trap-cooldown-from-socketed"
	env := portFixtureEnv(t, fixture, OutputModeMain)
	wantStat(t, fixture, env.Player.Output, "TrapCooldown", 2.013)
}

// TestPermanentMinionsHaveNoDuration pins CalcOffence.lua:1507.
func TestPermanentMinionsHaveNoDuration(t *testing.T) {
	const fixture = "skeleton-permanent"
	env := portFixtureEnv(t, fixture, OutputModeMain)
	wantAbsent(t, fixture, env.Player.Output, "Duration")
	wantStat(t, fixture, env.Player.Output, "DurationMod", 1)
}

// TestEnemyBlockChanceReductions pins CalcOffence.lua:2267-2270.
func TestEnemyBlockChanceReductions(t *testing.T) {
	for fixture, hitChance := range map[string]float64{"enemy-block-reduced": 21.6, "enemy-cannot-block": 24} {
		env := portFixtureEnv(t, fixture, OutputModeMain)
		wantStat(t, fixture, env.Player.Output, "HitChance", hitChance)
	}
}

// TestImpaleChanceOnlyInEffectiveMode pins CalcOffence.lua:4312.
func TestImpaleChanceOnlyInEffectiveMode(t *testing.T) {
	env := portFixtureEnv(t, "impale-unbuffed", OutputModeCalcs)
	wantStat(t, "impale-unbuffed", env.Player.Output, "ImpaleChance", 0)
	env = portFixtureEnv(t, "impale-effective", OutputModeCalcs)
	wantStat(t, "impale-effective", env.Player.Output, "ImpaleChance", 40)
	wantStat(t, "impale-effective", env.Player.Output, "ImpaleDPS", 0.8290408275)
}

// TestBurningGroundFromIgnite pins CalcOffence.lua:5276.
func TestBurningGroundFromIgnite(t *testing.T) {
	const fixture = "alchemists-mark-grounds"
	env := portFixtureEnv(t, fixture, OutputModeMain)
	wantStat(t, fixture, env.Player.Output, "BurningGroundFromIgnite", 1)
	wantStat(t, fixture, env.Player.Output, "BurningGroundDPS", 322.40775947396)
	wantStat(t, fixture, env.Player.Output, "TotalDotDPS", 1562.4376036046)
	wantAbsent(t, fixture, env.Player.Output, "CausticGroundFromPoison")
}

// TestImpaleBreakdownSkipsUnitDpsMultiplier pins CalcOffence.lua:6252-6254.
func TestImpaleBreakdownSkipsUnitDpsMultiplier(t *testing.T) {
	const fixture = "impale-effective"
	env := portFixtureEnv(t, fixture, OutputModeCalcs)
	entry := env.Player.Breakdown.Data["ImpaleDPS"]
	if entry == nil {
		t.Fatalf("REFUSING: %s has no ImpaleDPS breakdown", fixture)
	}
	want := []string{
		"52.44 (average physical hit before mitigation)",
		"x 0.24 (chance to hit)",
		"x 0.05 (impale damage multiplier)",
		"x 1.32 (attack rate)",
		"= 0.8",
	}
	got := make([]string, 0, len(entry.Lines))
	for _, line := range entry.Lines {
		got = append(got, strings.ReplaceAll(line, "^8", ""))
	}
	if !slices.Equal(got, want) {
		t.Errorf("%s: ImpaleDPS breakdown = %q, PoB %q", fixture, got, want)
	}
}

// TestEnemyResistEqualToPartyMember pins CalcOffence.lua:500-501.
func TestEnemyResistEqualToPartyMember(t *testing.T) {
	const fixture = "party-fire-resist-equal"
	env := portFixtureEnv(t, fixture, OutputModeMain)
	wantStat(t, fixture, env.Player.Output, "AverageHit", 3136.705)
	wantStat(t, fixture, env.Player.Output, "TotalDPS", 4182.2733333333)
	wantStat(t, fixture, env.Player.Output, "IgniteDPS", 3224.0775947396)
}

// TestOffenceSkillFlagsMatchPoB pins CalcOffence.lua:522-525 (SkillIsFocused reads chanceToTriggerOnFocus) and
// 2525-2527 (channelRelease). The wants are the pinned PoB's dumped skillFlags.
func TestOffenceSkillFlagsMatchPoB(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.30")
	for _, tc := range []struct {
		build string
		flag  SkillFlag
	}{
		{"4788-ladder-GeordiMcSnuff_1894.GeordiFlameOne", SkillFlagFocused},
		{"10850-ladder-MTMorten_3552.LetsTryThisFlameBow", SkillFlag("channelRelease")},
		{"15080-ladder-whysaygoodbyex_4369.WhySayGoodByedxawsd", SkillFlag("channelRelease")},
	} {
		ms := corpusEnvForLoadedData(t, tc.build).Player.MainSkill
		if ms == nil {
			t.Fatalf("REFUSING: %s has no main skill", tc.build)
		}
		if !ms.SkillFlags[tc.flag] {
			t.Errorf("%s: %s unset, PoB sets it", tc.build, tc.flag)
		}
	}
}
