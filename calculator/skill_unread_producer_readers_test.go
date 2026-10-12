package calculator

import (
	"math"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

func skillFuncFixture(t *testing.T, id, callback string, part int) (*ActiveSkill, skillFunc) {
	t.Helper()
	funcs := skillFuncsFor(id)
	if funcs == nil {
		t.Fatalf("REFUSING: skillFuncsFor(%q) is nil — the granted-effect id is not registered", id)
	}
	fn := funcs[callback]
	if fn == nil {
		t.Fatalf("REFUSING: %q has no %s", id, callback)
	}
	return &ActiveSkill{
		SkillFlags:   map[SkillFlag]bool{},
		SkillModList: moddb.NewModList(),
		SkillCfg:     &moddb.ListCfg{SkillPart: part},
		SkillData:    &SkillData{},
		SkillPart:    part,
	}, fn
}

// TestStaticStrikeBeamIntervalReadsStaticStrikeFrequency: StaticStrike's beam interval divides repeatFrequency by the zap-speed INCREASE.
func TestStaticStrikeBeamIntervalReadsStaticStrikeFrequency(t *testing.T) {
	t.Parallel()

	skill, fn := skillFuncFixture(t, "StaticStrike", "preDamageFunc", 2)
	skill.SkillData.RepeatFrequency = 0.6
	skill.SkillModList.AddMod(mod.NewFloat("StaticStrikeFrequency", mod.TypeIncrease, 50))

	fn(skill, map[string]float64{}, nil)

	want := 0.6 / 1.5
	if math.Abs(skill.SkillData.HitTimeOverride-want) > 1e-9 {
		t.Errorf("HitTimeOverride = %v, act_str.lua:10665 gives %v", skill.SkillData.HitTimeOverride, want)
	}

	bare, fn2 := skillFuncFixture(t, "StaticStrike", "preDamageFunc", 2)
	bare.SkillData.RepeatFrequency = 0.6
	fn2(bare, map[string]float64{}, nil)
	if bare.SkillData.HitTimeOverride != 0.6 {
		t.Errorf("with no StaticStrikeFrequency: HitTimeOverride = %v, want 0.6", bare.SkillData.HitTimeOverride)
	}
}

// TestStaticStrikePart1WritesNoHitTimeOverride: Part 1 carries no beam, so the callback must write nothing at all.
func TestStaticStrikePart1WritesNoHitTimeOverride(t *testing.T) {
	t.Parallel()

	skill, fn := skillFuncFixture(t, "StaticStrike", "preDamageFunc", 1)
	skill.SkillData.RepeatFrequency = 0.6
	skill.SkillModList.AddMod(mod.NewFloat("StaticStrikeFrequency", mod.TypeIncrease, 50))

	fn(skill, map[string]float64{}, nil)

	if skill.SkillData.HitTimeOverride != 0 {
		t.Errorf("part 1 HitTimeOverride = %v, want 0 — the Lua guards on skillPart == 2",
			skill.SkillData.HitTimeOverride)
	}
}

// TestStormRainTransfiguresReadAllowedStormArrows: The two Storm Rain transfigures have their OWN preDamageFunc bodies.
func TestStormRainTransfiguresReadAllowedStormArrows(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		id            string
		allowedArrows float64
		wantDpsMult   float64
	}{
		{"StormRainAltX", 2, 1},
		{"StormRainAltY", 2, 2},
	} {
		skill, fn := skillFuncFixture(t, tc.id, "preDamageFunc", 2)
		skill.SkillData.HitFrequency = 0.25
		skill.SkillModList.AddMod(mod.NewFloat("StormRainBeamFrequency", mod.TypeIncrease, 25))
		skill.SkillModList.AddMod(mod.NewFloat("StormRainAllowedStormArrows", mod.TypeBase, tc.allowedArrows))

		fn(skill, map[string]float64{}, nil)

		wantHit := 0.25 / 1.25
		if math.Abs(skill.SkillData.HitTimeOverride-wantHit) > 1e-9 {
			t.Errorf("%s: HitTimeOverride = %v, want %v", tc.id, skill.SkillData.HitTimeOverride, wantHit)
		}
		if math.Abs(skill.SkillData.DpsMultiplier-tc.wantDpsMult) > 1e-9 {
			t.Errorf("%s: DpsMultiplier = %v, want %v", tc.id, skill.SkillData.DpsMultiplier, tc.wantDpsMult)
		}
	}
}

// TestTornadoShotPart2FoldsSecondaryProjectiles: Tornado Shot's part 2 folds the secondary projectiles into dpsMultiplier (act_dex.lua:16048-16059).
func TestTornadoShotPart2FoldsSecondaryProjectiles(t *testing.T) {
	t.Parallel()

	skill, fn := skillFuncFixture(t, "TornadoShot", "preDamageFunc", 2)
	skill.SkillModList.AddMod(mod.NewFloat("tornadoShotSecondaryProjectiles", mod.TypeBase, 3))

	fn(skill, map[string]float64{"ProjectileCount": 4}, nil)

	want := 1 * (1 + 0.6*4)
	if math.Abs(skill.SkillData.DpsMultiplier-want) > 1e-9 {
		t.Errorf("DpsMultiplier = %v, act_dex.lua:16057 gives %v", skill.SkillData.DpsMultiplier, want)
	}

	barrage, fn2 := skillFuncFixture(t, "TornadoShot", "preDamageFunc", 2)
	barrage.SkillModList.AddMod(mod.NewFloat("tornadoShotSecondaryProjectiles", mod.TypeBase, 3))
	barrage.SkillModList.AddMod(mod.NewFlag("SequentialProjectiles", true))
	fn2(barrage, map[string]float64{"ProjectileCount": 4}, nil)
	wantBarrage := 1 * (1 + 0.6*1)
	if math.Abs(barrage.SkillData.DpsMultiplier-wantBarrage) > 1e-9 {
		t.Errorf("barrage DpsMultiplier = %v, want %v", barrage.SkillData.DpsMultiplier, wantBarrage)
	}
}

// TestFireAtMostOneProjectileParses: ModParser.lua:4528 is the pin's ONLY producer of SingleProjectile.
func TestFireAtMostOneProjectileParses(t *testing.T) {
	t.Parallel()

	e := ParseMod("Fire at most 1 Projectile", false)
	if e == nil || len(e.ModList) != 1 {
		t.Fatalf(`ParseMod("Fire at most 1 Projectile") produced %v, want one SingleProjectile flag`, e)
	}
	if e.ModList[0].Name() != "SingleProjectile" {
		t.Errorf("mod name = %q, want SingleProjectile", e.ModList[0].Name())
	}
	if e.Extra != "" {
		t.Errorf("unparsed remainder %q — PoB parses the line WHOLE, and a remainder discards it", e.Extra)
	}
}
