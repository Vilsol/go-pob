package calculator

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data"
)

// TestWalkDrainMatchesReducePoolsByDamage grades numberOfHitsToDie's INLINE drain against ReducePoolsByDamage, hit by hit, on the same inputs.
func TestWalkDrainMatchesReducePoolsByDamage(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	corpus := corpusDirForLoadedData(t)
	files, err := filepath.Glob(filepath.Join(corpus, "*.xml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("REFUSING: %s holds no builds (%v)", corpus, err)
	}
	if limit := envInt("LIMIT", 0); limit > 0 && len(files) > limit {
		files = files[:limit]
	}

	eligible, steps, diverging, wardDiverging := 0, 0, 0, 0
	byPool := map[string]int{}
	divBuilds := map[string]struct{}{}
	divWithWard, divWithoutWard := 0, 0
	wardBuilds := map[string]struct{}{}
	wardNonzeroOut := 0
	carry := map[string]int{}
	carryBuilds := map[string]map[string]struct{}{
		"aegis": {}, "guard": {}, "ally": {}, "ward": {}, "minion": {},
	}
	for _, f := range files {
		xml, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		b, err := builds.ParseBuild(xml)
		if err != nil {
			continue
		}
		calc := &Calculator{PoB: b}
		env, _, _, _ := InitEnv(calc.PoB, corpusCacheForTest(t), OutputModeMain)
		env.TracePoolWalk = true
		PerformCalc(env)

		if len(env.PoolWalkTrace) == 0 {
			continue
		}
		eligible++
		reported := false

		for _, step := range env.PoolWalkTrace {
			steps++
			actor := step.Actor
			if actor == nil {
				t.Fatalf("%s hit %d: the trace recorded no actor, so there is no output table to "+
					"reconstruct against — REFUSING rather than guessing the player",
					filepath.Base(f), step.N)
			}
			base := filepath.Base(f)
			if env.Minion != nil && actor == env.Minion.Actor {
				carry["minion"]++
				carryBuilds["minion"][base] = struct{}{}
			}
			if anyNonzero(step.AegisIn) {
				carry["aegis"]++
				carryBuilds["aegis"][base] = struct{}{}
			}
			if anyNonzero(step.GuardIn) {
				carry["guard"]++
				carryBuilds["guard"][base] = struct{}{}
			}
			if anyNonzero(step.AllyIn) {
				carry["ally"]++
				carryBuilds["ally"][base] = struct{}{}
			}
			if step.WardIn > 0 {
				carry["ward"]++
				carryBuilds["ward"][base] = struct{}{}
			}
			damage := make(map[data.DamageType]float64, len(step.Damage))
			for dt, v := range step.Damage {
				damage[dt] = v
			}
			allies := make(map[string]*PoolState, len(step.AllyIn))
			for _, ally := range allyLifePoolList {
				remaining, ok := step.AllyIn[ally.key]
				if !ok {
					continue
				}
				allies[ally.key] = &PoolState{
					Remaining: remaining,
					Percent:   math.Min(actor.Output[ally.mitigation], 100) / 100,
				}
			}
			got := ReducePoolsByDamage(&PoolTable{
				AlliesTakenBeforeYou:         allies,
				Aegis:                        cloneStrF(step.AegisIn),
				Guard:                        cloneStrF(step.GuardIn),
				Ward:                         ptrF(step.WardIn),
				EnergyShield:                 ptrF(step.EnergyShieldIn),
				Mana:                         ptrF(step.ManaIn),
				Life:                         ptrF(step.LifeIn),
				DamageTakenThatCanBeRecouped: map[data.DamageType]float64{},
			}, damage, actor)

			cmp := []struct {
				name string
				got  float64
				want float64
			}{
				{"Life", math.Max(step.LifeOut, 0), *got.Life},
				{"Mana", math.Max(step.ManaOut, 0), *got.Mana},
				{"EnergyShield", math.Max(step.EnergyShieldOut, 0), *got.EnergyShield},
			}
			for _, k := range sortedKeys(step.AegisOut) {
				cmp = append(cmp, struct {
					name string
					got  float64
					want float64
				}{"Aegis[" + k + "]", step.AegisOut[k], got.Aegis[k]})
			}
			for _, k := range sortedKeys(step.GuardOut) {
				cmp = append(cmp, struct {
					name string
					got  float64
					want float64
				}{"Guard[" + k + "]", step.GuardOut[k], got.Guard[k]})
			}
			for _, k := range sortedKeys(step.AllyOut) {
				want := math.NaN()
				if p := got.AlliesTakenBeforeYou[k]; p != nil {
					want = p.Remaining
				}
				cmp = append(cmp, struct {
					name string
					got  float64
					want float64
				}{"Ally[" + k + "]", step.AllyOut[k], want})
			}
			for _, p := range cmp {
				if math.Abs(p.got-p.want) > 1e-9 {
					diverging++
					byPool[p.name]++
					divBuilds[filepath.Base(f)] = struct{}{}
					wardHeld := step.WardIn > 0 || step.WardOut > 0 || *got.Ward > 0
					if wardHeld {
						divWithWard++
					} else {
						divWithoutWard++
					}
					if reported && wardHeld {
						break
					}
					reported = true
					logf := t.Logf
					if !wardHeld {
						logf = t.Errorf
					}
					logf("%s hit %d: the walk's inline drain left %s = %v where "+
						"ReducePoolsByDamage leaves %v — ONE rule, TWO implementations, and they "+
						"disagree.\n    in   L%v M%v ES%v W%v aegis%v guard%v ally%v   incoming %v\n"+
						"    walk L%v M%v ES%v W%v aegis%v guard%v ally%v\n"+
						"    rpbd L%v M%v ES%v\n    plt=%v sharedMoM=%v EBflag=%v%v",
						filepath.Base(f), step.N, p.name, p.got, p.want,
						step.LifeIn, step.ManaIn, step.EnergyShieldIn, step.WardIn,
						step.AegisIn, step.GuardIn, step.AllyIn, step.Damage,
						step.LifeOut, step.ManaOut, step.EnergyShieldOut, step.WardOut,
						step.AegisOut, step.GuardOut, step.AllyOut,
						*got.Life, *got.Mana, *got.EnergyShield,
						actor.Output["preventedLifeLossTotal"], actor.Output["sharedMindOverMatter"],
						actor.ModDB.Flag(nil, "EnergyShieldProtectsMana"), perTypeDbg(actor))
					break
				}
			}
			if step.WardOut != 0 || *got.Ward != 0 {
				wardNonzeroOut++
			}
			if math.Abs(step.WardOut-*got.Ward) > 1e-9 {
				wardDiverging++
				wardBuilds[filepath.Base(f)] = struct{}{}
			}
			if divWithoutWard > envInt("MAXDIVBUILDS", 40) {
				t.Fatalf("stopping after %d ward-free diverging steps — a wall of these hides "+
					"the SECOND cause behind the first (by pool: %v)", divWithoutWard, byPool)
			}
		}
	}

	if eligible == 0 {
		t.Fatalf("REFUSING: none of %d builds produced a walk trace, so nothing was compared. "+
			"Empty must mean 'I looked and found nothing', never 'I did not look'", len(files))
	}
	t.Logf("=== WALK DRAIN vs ReducePoolsByDamage === %d of %d builds walked, %d hits compared, "+
		"%d diverging on %d builds (by pool: %v); ward, counted apart because the two sides hold different "+
		"RULES there: %d steps on %d builds (%d steps had a nonzero ward on either side — a 0 "+
		"there means nothing to compare, not agreement)",
		eligible, len(files), steps, diverging, len(divBuilds), byPool, wardDiverging,
		len(wardBuilds), wardNonzeroOut)
	for _, k := range []string{"aegis", "guard", "ally", "ward", "minion"} {
		t.Logf("    carrying %-7s %6d steps on %5d builds", k, carry[k], len(carryBuilds[k]))
	}
	t.Logf("    diverging: %d carrying ward, %d NOT", divWithWard, divWithoutWard)

	if divWithoutWard > 0 {
		t.Errorf("%d diverging steps carry NO ward — that is not the known rule gap and the "+
			"reports above name them", divWithoutWard)
	}
	if diverging > walkDrainWardDivergenceCeiling {
		t.Errorf("%d diverging steps on %d builds, ceiling %d — the two drains have grown further "+
			"apart, not closer", diverging, len(divBuilds), walkDrainWardDivergenceCeiling)
	}
}

const walkDrainWardDivergenceCeiling = 0

func anyNonzero(m map[string]float64) bool {
	for _, v := range m {
		if v != 0 {
			return true
		}
	}
	return false
}

func cloneStrF(m map[string]float64) map[string]float64 {
	out := make(map[string]float64, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func perTypeDbg(actor *Actor) string {
	out := ""
	for _, dt := range data.DamageType("").Values() {
		out += fmt.Sprintf(" %s(mom=%v byp=%v aeg=%v grd=%v)", dt,
			actor.Output[string(dt)+"MindOverMatter"], actor.Output[string(dt)+"EnergyShieldBypass"],
			actor.Output[string(dt)+"Aegis"], actor.Output[string(dt)+"GuardAbsorb"])
	}
	return out
}
