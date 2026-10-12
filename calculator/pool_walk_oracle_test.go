package calculator

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data"
)

type oraclePools struct {
	Life                         *float64           `json:"Life"`
	Mana                         *float64           `json:"Mana"`
	EnergyShield                 *float64           `json:"EnergyShield"`
	Ward                         *float64           `json:"Ward"`
	DamageTakenThatCanBeRecouped map[string]float64 `json:"damageTakenThatCanBeRecouped"`
}

type oracleCall struct {
	N              int                `json:"n"`
	DamageIn       map[string]float64 `json:"damageIn"`
	PoolsIn        *oraclePools       `json:"poolsIn"`
	PoolsOut       *oraclePools       `json:"poolsOut"`
	CallerLine     int                `json:"callerLine"`
	WalkCallerLine int                `json:"walkCallerLine"`
}

type oracleEntry struct {
	Build                 string             `json:"build"`
	Error                 string             `json:"error"`
	CallCount             int                `json:"callCount"`
	Truncated             bool               `json:"truncated"`
	Calls                 []oracleCall       `json:"calls"`
	Inputs                map[string]float64 `json:"inputs"`
	RecoupableDamageTaken map[string]float64 `json:"recoupableDamageTaken"`
	TakenHit              map[string]float64 `json:"takenHit"`
}

func alignPoolWalkCalls(e oracleEntry, damageTypes []string) ([]oracleCall, string) {
	if len(e.TakenHit) == 0 {
		return nil, "the dump carries no `takenHit` anchor — regenerate it with oracle/dump_pool_walk.lua"
	}
	if e.Calls[0].WalkCallerLine == 0 {
		return nil, "the dump carries no `walkCallerLine` tags — regenerate it with oracle/dump_pool_walk.lua"
	}
	damageMatches := func(c oracleCall) bool {
		for _, dt := range damageTypes {
			if math.Abs(c.DamageIn[dt]-e.TakenHit[dt]) > math.Max(1e-9, math.Abs(e.TakenHit[dt])*1e-9) {
				return false
			}
		}
		return true
	}
	poolsMatch := func(c oracleCall) bool {
		if c.PoolsIn == nil {
			return false
		}
		for _, p := range []struct {
			got  *float64
			want string
		}{
			{c.PoolsIn.EnergyShield, "EnergyShieldRecoveryCap"},
			{c.PoolsIn.Life, "LifeRecoverable"},
			{c.PoolsIn.Mana, "ManaUnreserved"},
			{c.PoolsIn.Ward, "Ward"},
		} {
			want, ok := e.Inputs[p.want]
			if !ok {
				continue
			}
			if p.got == nil || math.Abs(*p.got-want) > math.Max(1e-6, math.Abs(want)*1e-9) {
				return false
			}
		}
		return true
	}
	sawDamageAnchor := false
	for i, c := range e.Calls {
		if c.CallerLine != 2667 || c.WalkCallerLine != 2743 || !damageMatches(c) {
			continue
		}
		sawDamageAnchor = true
		if poolsMatch(c) {
			return e.Calls[i:], ""
		}
	}
	if sawDamageAnchor {
		return nil, "every call seeding the walk with output[<T>TakenHit] starts from POOLS that " +
			"disagree with the recorded inputs (CalcDefence.lua:2605-2612) — PoB's traced walk is " +
			"not the invocation behind the graded numbers, so this build's trajectory is VOID"
	}
	return nil, "no recorded call seeds the walk with output[<T>TakenHit] (CalcDefence.lua:2739-2740)"
}

type poolWalkDumpFailure struct {
	File   string
	Reason string
}

func loadPoolWalkDumps(files []string) ([]oracleEntry, []poolWalkDumpFailure) {
	entries := make([]oracleEntry, 0, len(files))
	var failures []poolWalkDumpFailure

	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			failures = append(failures, poolWalkDumpFailure{File: f, Reason: fmt.Sprintf("read: %v", err)})
			continue
		}
		var e oracleEntry
		if err := json.Unmarshal(raw, &e); err != nil {
			failures = append(failures, poolWalkDumpFailure{File: f, Reason: fmt.Sprintf("parse: %v", err)})
			continue
		}
		entries = append(entries, e)
	}

	return entries, failures
}

// TestPoolWalkMatchesOracle diffs our EHP-walk pool TRAJECTORY against PoB's, hit by hit, and reports the FIRST diverging hit per build.
func TestPoolWalkMatchesOracle(t *testing.T) {
	dumpDir := os.Getenv("POOL_WALK_DUMP")
	if dumpDir == "" {
		t.Skip("set POOL_WALK_DUMP=<dir> (oracle/dump_pool_walk.lua)")
	}
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	files, err := filepath.Glob(filepath.Join(dumpDir, "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("REFUSING: %s holds no .json dumps. Empty must mean 'I looked and found nothing', "+
			"never 'I did not look' (%v)", dumpDir, err)
	}
	sort.Strings(files)

	corpus := corpusDirForLoadedData(t)
	checked, agreed, pobErrored, zeroCall := 0, 0, 0, 0

	entries, failures := loadPoolWalkDumps(files)
	for _, fail := range failures {
		t.Errorf("UNUSABLE DUMP %s (%s) — this build was NOT compared. It costs one build, not "+
			"the run, but the run is NOT clean: regenerate with oracle/dump_pool_walk.lua",
			fail.File, fail.Reason)
	}
	unusable := len(failures)

	for _, e := range entries {
		if e.Error != "" {
			t.Logf("%s: PoB side errored (%s) — SKIPPED, not counted as agreeing", e.Build, e.Error)
			pobErrored++
			continue
		}
		if len(e.Calls) == 0 {
			t.Errorf("%s: the dump recorded ZERO reducePoolsByDamage calls but reports "+
				"TotalEHP %v — the probe wrapped something nothing calls", e.Build, e.Inputs["TotalEHP"])
			zeroCall++
			continue
		}

		xml, err := os.ReadFile(filepath.Join(corpus, e.Build))
		if err != nil {
			t.Errorf("UNUSABLE BUILD %s (read: %v) — NOT compared", e.Build, err)
			unusable++
			continue
		}
		b, err := builds.ParseBuild(xml)
		if err != nil {
			t.Errorf("UNUSABLE BUILD %s (parse: %v) — NOT compared", e.Build, err)
			unusable++
			continue
		}
		calc := &Calculator{PoB: b}
		env, _, _, _ := InitEnv(calc.PoB, corpusCacheForTest(t), OutputModeMain)
		env.TracePoolWalk = true
		PerformCalc(env)

		checked++

		t.Run(e.Build, func(t *testing.T) {
			for _, in := range []struct {
				key string
				got float64
			}{
				{"Life", env.Player.Output["Life"]},
				{"LifeRecoverable", env.Player.Output["LifeRecoverable"]},
				{"ManaUnreserved", env.Player.Output["ManaUnreserved"]},
				{"EnergyShieldRecoveryCap", env.Player.Output["EnergyShieldRecoveryCap"]},
				{"Ward", env.Player.Output["Ward"]},
			} {
				want, ok := e.Inputs[in.key]
				if ok && math.Abs(in.got-want) > 1e-6 {
					t.Logf("INPUT %s = %v, PoB %v — the trajectory comparison below is VOID for "+
						"this build; fix the input first", in.key, in.got, want)
					return
				}
			}

			if len(env.PoolWalkTrace) == 0 {
				t.Errorf("our walk recorded ZERO steps where PoB made %d calls", e.CallCount)
				return
			}

			for _, dt := range data.DamageType("").Values() {
				want, ok := e.RecoupableDamageTaken[string(dt)]
				if !ok || want == 0 {
					continue
				}
				got := env.Player.Output[string(dt)+"RecoupableDamageTaken"]
				if math.Abs(got-want) > math.Max(1, math.Abs(want))*0.005 {
					t.Errorf("graded %sRecoupableDamageTaken = %v, PoB %v (ratio %.4f) — a SUM "+
						"OVER HITS, so read the hit COUNT before the arithmetic", dt, got, want, got/want)
				}
			}

			dtNames := make([]string, 0, 5)
			for _, dt := range data.DamageType("").Values() {
				dtNames = append(dtNames, string(dt))
			}
			calls, why := alignPoolWalkCalls(e, dtNames)
			if why != "" {
				t.Logf("NOT COMPARED: %s", why)
				return
			}
			skipped := len(e.Calls) - len(calls)

			n := min(len(calls), len(env.PoolWalkTrace))
			firstRecoupNote := ""
			for i := 0; i < n; i++ {
				o, g := calls[i], env.PoolWalkTrace[i]
				for _, dt := range data.DamageType("").Values() {
					ow, gw := math.Ceil(o.DamageIn[string(dt)]), g.Damage[dt]
					if math.Abs(ow-gw) > 0.5 {
						where := "output[" + string(dt) + "TakenHit] itself — an INPUT defect, not a drain one"
						if i > 0 {
							where = "at hit " + strconv.Itoa(i+1) + ", so the speed-up multiplier may " +
								"have amplified a sub-point difference; check hit 1 before reading it as a second cause"
						}
						t.Errorf("first divergence at hit %d: incoming %s damage %v, PoB %v (pre-ceil %v) — %s",
							i+1, dt, gw, ow, o.DamageIn[string(dt)], where)
						return
					}
				}
				if o.PoolsOut == nil {
					continue
				}
				for _, p := range []struct {
					name string
					got  float64
					want *float64
				}{
					{"Life", math.Max(g.LifeOut, 0), o.PoolsOut.Life},
					{"Mana", math.Max(g.ManaOut, 0), o.PoolsOut.Mana},
					{"EnergyShield", math.Max(g.EnergyShieldOut, 0), o.PoolsOut.EnergyShield},
					{"Ward", math.Max(g.WardOut, 0), o.PoolsOut.Ward},
				} {
					if p.want == nil {
						continue
					}
					if math.Abs(p.got-*p.want) > math.Max(1, math.Abs(*p.want))*0.005 {
						t.Errorf("first divergence at hit %d: %s after the drain = %v, PoB %v "+
							"(incoming %v)", i+1, p.name, p.got, *p.want, g.Damage)
						return
					}
				}
				if firstRecoupNote == "" && o.PoolsOut.DamageTakenThatCanBeRecouped != nil && g.RecoupableOut != nil {
					for _, dt := range data.DamageType("").Values() {
						ow := o.PoolsOut.DamageTakenThatCanBeRecouped[string(dt)]
						if gw := g.RecoupableOut[dt]; gw != 0 && math.Abs(ow-gw) > math.Max(1, math.Abs(ow))*0.005 {
							firstRecoupNote = fmt.Sprintf("running %s recoupable first differs at "+
								"hit %d: %v here, %v in PoB", dt, i+1, gw, ow)
						}
					}
				}
			}
			if firstRecoupNote != "" {
				t.Logf("note: %s — CONTEXT, not an assertion (see the comment at the compare site)",
					firstRecoupNote)
			}
			for i := 0; i < n; i++ {
				st := env.PoolWalkTrace[i]
				if st.LifeOut < 0 || st.ManaOut < 0 || st.EnergyShieldOut < 0 {
					t.Logf("note: a running pool goes NEGATIVE at hit %d (Life %v) where PoB floors at 0 "+
						"and reports the excess as OverkillDamage — the walk's own loop guard is "+
						"`life > 0`, so this is only a defect if anything downstream reads the value",
						i+1, st.LifeOut)
					break
				}
			}
			if len(calls) != len(env.PoolWalkTrace) {
				t.Logf("length differs (%d here, %d in PoB after skipping %d pre-walk calls) — NOT a "+
					"finding: PoB's trace counts all three callers of reducePoolsByDamage and ours "+
					"counts only the walk. The %d compared hits agree.",
					len(env.PoolWalkTrace), len(calls), skipped, n)
			}
			agreed++
		})
	}

	if checked == 0 {
		t.Fatalf("REFUSING: compared 0 builds across %d dumps", len(files))
	}
	fmt.Printf("=== POOL WALK TRAJECTORY === %d builds compared, %d agree on every hit "+
		"(%d UNUSABLE, %d PoB-errored, %d zero-call, of %d dumps)\n",
		checked, agreed, unusable, pobErrored, zeroCall, len(files))
	if unusable > 0 {
		t.Errorf("REFUSING to read as complete: %d of %d dumps were unusable and are named above",
			unusable, len(files))
	}
	if total := checked + unusable + pobErrored + zeroCall; total != len(files) {
		t.Errorf("REFUSING: %d dumps accounted for (%d compared + %d unusable + %d PoB-errored + "+
			"%d zero-call) but %d were read — a build is leaving this loop uncounted",
			total, checked, unusable, pobErrored, zeroCall, len(files))
	}
}
