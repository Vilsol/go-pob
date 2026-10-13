package calculator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds/parsecache"
)

// TestCalcsIntermediateCensus ranks the FIRST diverging CALCS-mode intermediate on every build oracle/dump_calcs_damage.lua dumped.
func TestCalcsIntermediateCensus(t *testing.T) {
	requireOperatorRun(t, "report-only census")
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true required")
	}
	dumpDir := os.Getenv("CALCS_DUMP")
	if dumpDir == "" {
		t.Skip("set CALCS_DUMP=<dir> from `task calcs-chain` (aligned CALCS side-car)")
	}
	wlDir := os.Getenv("WL")
	if wlDir == "" {
		t.Fatal("REFUSING: set WL=<dir> written by `WORKLIST=<dir> task gate`")
	}
	requireGameData(t, "3.29")

	wrong, err := readWorklistWrong(filepath.Join(wlDir, "builds.tsv"))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	files, err := filepath.Glob(filepath.Join(dumpDir, "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("REFUSING: %s holds no .json dumps (%v)", dumpDir, err)
	}
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	guardPicks := loadGuardPicks(t)
	pairsPicks := loadPairsPicks(t)
	corpus := corpusDirForLoadedData(t)

	type result struct {
		build, skipped          string
		rows                    []intermediateDiff
		unclassified, unwritten []string
		nulls, graded           int
	}
	results := make([]result, len(files))

	runConcurrently(len(files), evalWorkers(), func(i int, cache *EnvironmentCache) {
		res := &results[i]
		raw, err := os.ReadFile(files[i])
		if err != nil {
			res.skipped = "dump unreadable"
			return
		}
		var r struct {
			Build         string           `json:"build"`
			Error         string           `json:"error"`
			SkillMismatch bool             `json:"skillMismatch"`
			MainSkill     string           `json:"mainSkill"`
			Output        intermediateNums `json:"output"`
			MainHand      intermediateNums `json:"outputMainHand"`
			OffHand       intermediateNums `json:"outputOffHand"`
		}
		if json.Unmarshal(raw, &r) != nil || r.Build == "" {
			res.skipped = "dump unparseable"
			return
		}
		res.build = r.Build
		rec, ok := oracle[r.Build]
		switch {
		case r.Error != "":
			res.skipped = "PoB errored in the dumper"
			return
		case r.SkillMismatch:
			res.skipped = "PoB CALCS skill != MAIN skill"
			return
		case !ok:
			res.skipped = "no oracle record"
			return
		case rec.TreeSources < oracleMinTreeSources:
			res.skipped = "excluded by the gate (broken tree)"
			return
		}
		xml, err := os.ReadFile(filepath.Join(corpus, r.Build))
		if err != nil {
			res.skipped = "corpus xml missing"
			return
		}
		b, err := parsecache.ParseCached(xml)
		if err != nil {
			res.skipped = "ParseBuild failed"
			return
		}
		cache.guardPick = guardPicks[r.Build]
		cache.pairsPicks = pairsPicks[r.Build]
		var env *Environment
		func() {
			defer func() { _ = recover() }()
			env = BuildOutputWithCache(&Calculator{PoB: b, AlignCalcsSkillToMain: true}, OutputModeCalcs, cache)
		}()
		cache.guardPick = ""
		cache.pairsPicks = nil
		if env == nil || env.Player == nil {
			res.skipped = "our CALCS run panicked"
			return
		}
		ours := ""
		if s := env.Player.MainSkill; s != nil && s.ActiveEffect != nil && s.ActiveEffect.GrantedEffect != nil {
			ours = s.ActiveEffect.GrantedEffect.Name()
		}
		if ours != r.MainSkill {
			res.skipped = "our CALCS skill != PoB's"
			return
		}
		res.nulls = r.Output.nulls + r.MainHand.nulls + r.OffHand.nulls
		res.rows, res.unclassified, res.unwritten = intermediateFirstDiffs([]intermediateTable{
			{"player", r.Output.m, env.Player.Output, rec.Output},
			{"MainHand", r.MainHand.m, env.Player.OutputTable[OutTableMainHand], rec.MainHand},
			{"OffHand", r.OffHand.m, env.Player.OutputTable[OutTableOffHand], rec.OffHand},
		})
		res.graded = len(r.Output.m) + len(r.MainHand.m) + len(r.OffHand.m) - len(res.unwritten)
	})

	skippedBy := map[string]int{}
	var compared []*result
	unclassified, unwritten := map[string]bool{}, map[string]int{}
	nulls := 0
	for i := range results {
		r := &results[i]
		if r.skipped != "" {
			skippedBy[r.skipped]++
			continue
		}
		compared = append(compared, r)
		nulls += r.nulls
		for _, k := range r.unclassified {
			unclassified[k] = true
		}
		for _, k := range r.unwritten {
			unwritten[k]++
		}
	}
	if len(compared) == 0 {
		t.Fatalf("REFUSING: 0 of %d dumps were comparable: %v", len(files), skippedBy)
	}

	comparedSet := map[string]bool{}
	for _, r := range compared {
		comparedSet[r.build] = true
	}
	wrongCompared, wrongPairs, wrongAll, notDumped := 0, 0, 0, 0
	for b, w := range wrong {
		wrongAll += w
		if comparedSet[b] {
			wrongCompared++
			wrongPairs += w
		}
	}
	dumped := map[string]bool{}
	for i := range results {
		dumped[results[i].build] = true
	}
	for b := range wrong {
		if !dumped[b] {
			notDumped++
		}
	}
	t.Logf("%d dumps; %d compared (%d wrong in WL, carrying %d of %d wrong pairs); %d WL builds not dumped",
		len(files), len(compared), wrongCompared, wrongPairs, wrongAll, notDumped)
	for _, why := range sortedCountKeys(skippedBy) {
		t.Logf("  skipped %4d  %s", skippedBy[why], why)
	}
	t.Logf("  not graded: %d PoB NaN/Inf values (dumped as null)", nulls)
	if len(unwritten) > 0 {
		t.Logf("  not graded: %d CALCS-only keys (absent from the MAIN oracle) that we never write, by builds: %v",
			len(unwritten), unwritten)
	}
	if len(unclassified) > 0 {
		t.Logf("  %d keys fell to %s: %v", len(unclassified), intermediateUnclassified, sortedBoolKeys(unclassified))
	}

	if out := os.Getenv("CENSUS_TSV"); out != "" {
		var sb strings.Builder
		sb.WriteString("build\tactor\tkind\tkey\tmissing\textra\n")
		for _, r := range compared {
			fmt.Fprintf(&sb, "%s\t-\t-\t-\t0\t0\n", r.build)
			for _, d := range r.rows {
				fmt.Fprintf(&sb, "%s\t%s\t%s\t%s\t%d\t%d\n", r.build, d.table, d.rung, d.key, d.missing(), d.extra())
			}
		}
		if err := os.WriteFile(out, []byte(sb.String()), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", out)
	}

	type agg struct {
		builds map[string]bool
		pairs  int
	}
	perRung := map[string]*agg{}
	clusters := map[string]*agg{}
	clean, cleanWrong, cleanMinGraded := 0, 0, math.MaxInt
	for _, r := range compared {
		if len(r.rows) == 0 {
			clean++
			cleanMinGraded = min(cleanMinGraded, r.graded)
			if wrong[r.build] > 0 {
				cleanWrong++
			}
		}
		seenRung, seenCluster := map[string]bool{}, map[string]bool{}
		for _, d := range r.rows {
			c := fmt.Sprintf("%s %s %s.%s", d.rung, d.dir(), d.table, d.key)
			for _, x := range []struct {
				m    map[string]*agg
				seen map[string]bool
				k    string
			}{{perRung, seenRung, d.rung}, {clusters, seenCluster, c}} {
				if x.seen[x.k] {
					continue
				}
				x.seen[x.k] = true
				if x.m[x.k] == nil {
					x.m[x.k] = &agg{builds: map[string]bool{}}
				}
				x.m[x.k].builds[r.build] = true
				x.m[x.k].pairs += wrong[r.build]
			}
		}
	}
	t.Logf("%d compared builds diverge nowhere (%d of them wrong in WL; fewest keys graded on one: %d)",
		clean, cleanWrong, cleanMinGraded)

	byPairs := func(m map[string]*agg) []string {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			if m[keys[i]].pairs != m[keys[j]].pairs {
				return m[keys[i]].pairs > m[keys[j]].pairs
			}
			return keys[i] < keys[j]
		})
		return keys
	}
	wrongOf := func(bs map[string]bool) int {
		n := 0
		for b := range bs {
			if wrong[b] > 0 {
				n++
			}
		}
		return n
	}
	base := float64(wrongCompared) / float64(len(compared))

	t.Logf("FIRST DIVERGING RUNG, per chain (a build counts once per chain):")
	t.Logf("%6s %6s %6s  %s", "PAIRS", "WRONGB", "DIVB", "RUNG")
	for _, k := range byPairs(perRung) {
		a := perRung[k]
		t.Logf("%6d %6d %6d  %s", a.pairs, wrongOf(a.builds), len(a.builds), k)
	}

	top := 25
	if n, err := strconv.Atoi(os.Getenv("TOP")); err == nil && n > 0 {
		top = n
	}
	groups, groupKeys := map[string]*agg{}, map[string][]string{}
	for _, k := range byPairs(clusters) {
		sig := strings.Join(sortedBoolKeys(clusters[k].builds), "\x00")
		if groups[sig] == nil {
			groups[sig] = clusters[k]
		}
		groupKeys[sig] = append(groupKeys[sig], k)
	}
	t.Logf("TOP %d of %d clusters ((first-rung key, direction) grouped by identical build set) BY RESIDUAL"+
		" — enrichment = wrong share / base rate %.2f:", top, len(groups), base)
	t.Logf("%6s %6s %6s %6s  %s  %s", "PAIRS", "ENRICH", "WRONGB", "DIVB", "RUNG DIR TABLE.KEY", "[top builds]")
	for i, sig := range byPairs(groups) {
		if i >= top {
			break
		}
		a := groups[sig]
		k := groupKeys[sig][0]
		if n := len(groupKeys[sig]); n > 1 {
			more := groupKeys[sig][1:min(n, 6)]
			for j := range more {
				more[j] = more[j][strings.LastIndex(more[j], " ")+1:]
			}
			k = fmt.Sprintf("%s (+%d: %s)", k, n-1, strings.Join(more, ","))
		}
		wb := wrongOf(a.builds)
		bl := make([]string, 0, len(a.builds))
		for b := range a.builds {
			if wrong[b] > 0 {
				bl = append(bl, b)
			}
		}
		sort.Slice(bl, func(i, j int) bool {
			return wrong[bl[i]] > wrong[bl[j]] || wrong[bl[i]] == wrong[bl[j]] && bl[i] < bl[j]
		})
		var tb []string
		for _, b := range bl[:min(4, len(bl))] {
			tb = append(tb, fmt.Sprintf("%s:%d", strings.SplitN(b, "-", 2)[0], wrong[b]))
		}
		enr := 0.0
		if base > 0 {
			enr = float64(wb) / float64(len(a.builds)) / base
		}
		t.Logf("%6d %6.1f %6d %6d  %s  [%s]", a.pairs, enr, wb, len(a.builds), k, strings.Join(tb, " "))
	}
}

// TestCalcsIntermediateFirstDiffs is the attributor's control: an agreeing build reports nothing.
func TestCalcsIntermediateFirstDiffs(t *testing.T) {
	clean := func() map[string]float64 {
		return map[string]float64{
			"Str": 100, "Life": 5000, "FireResist": 75, "TotalEHP": 40000,
			"FireMinBase": 100, "FireEffMult": 1.5, "FireMin": 300, "FireMax": 600,
			"CritChance": 50, "FireHitAverage": 450, "AverageHit": 450,
			"IgniteDPS": 900, "TotalDPS": 4000, "ManaCost": 10, "LifeRegen": 50,
		}
	}
	names := func(ds []intermediateDiff) []string {
		out := make([]string, 0, len(ds))
		for _, d := range ds {
			out = append(out, d.rung+" "+d.dir()+" "+d.key)
		}
		sort.Strings(out)
		return out
	}

	for _, tc := range []struct {
		name   string
		break_ func(got map[string]float64)
		want   []string
	}{
		{"agreeing build reports nothing", func(map[string]float64) {}, nil},
		{"wrong base names only the base", func(g map[string]float64) {
			g["FireMinBase"] = 50
			g["FireMin"] = 150
			g["FireHitAverage"] = 375
			g["AverageHit"] = 375
			g["IgniteDPS"] = 700
			g["TotalDPS"] = 3000
		}, []string{"offence.2-base MISS FireMinBase"}},
		{"clean base, wrong Min/Max names the modifier rung", func(g map[string]float64) {
			g["FireMin"] = 450
			g["FireMax"] = 900
			g["FireHitAverage"] = 675
			g["TotalDPS"] = 6000
		}, []string{"offence.4-minmax XTRA FireMax", "offence.4-minmax XTRA FireMin"}},
		{"chains are independent: pools and dps each name their own first rung", func(g map[string]float64) {
			g["Life"] = 4000
			g["TotalEHP"] = 30000
			g["TotalDPS"] = 1
		}, []string{"defence.1-pools MISS Life", "offence.8-dps MISS TotalDPS"}},
		{"a key PoB lacks is not graded", func(g map[string]float64) {
			g["SomethingOnlyWeEmit"] = 5
		}, nil},
		{"a MAIN-graded key we lack reads as ours 0", func(g map[string]float64) {
			delete(g, "LifeRegen")
		}, []string{"recovery.1-recovery MISS LifeRegen"}},
		{"a CALCS-only key we never write is not graded", func(g map[string]float64) {
			delete(g, "FireMinBase")
		}, nil},
		{"a CALCS-only key we write is graded", func(g map[string]float64) {
			g["FireMinBase"] = 0
		}, []string{"offence.2-base MISS FireMinBase"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := clean()
			tc.break_(got)
			graded := clean()
			delete(graded, "FireMinBase")
			diffs, _, _ := intermediateFirstDiffs([]intermediateTable{{"player", clean(), got, graded}})
			if g := names(diffs); strings.Join(g, "|") != strings.Join(tc.want, "|") {
				t.Errorf("got %v, want %v", g, tc.want)
			}
		})
	}

	t.Run("NaN is both directions", func(t *testing.T) {
		got := clean()
		got["Str"] = math.NaN()
		diffs, _, _ := intermediateFirstDiffs([]intermediateTable{{"player", clean(), got, clean()}})
		if len(diffs) != 1 || diffs[0].missing() != 1 || diffs[0].extra() != 1 {
			t.Errorf("got %+v", diffs)
		}
	})

	t.Run("every key in the clean fixture is classified", func(t *testing.T) {
		for k := range clean() {
			if r := intermediateRungOf(k); r.chain == intermediateUnclassified {
				t.Errorf("%s is unclassified", k)
			}
		}
	})
}

type intermediateTable struct {
	name              string
	want, got, graded map[string]float64
}

type intermediateNums struct {
	m     map[string]float64
	nulls int
}

func (n *intermediateNums) UnmarshalJSON(b []byte) error {
	if s := strings.TrimSpace(string(b)); s == "null" || strings.HasPrefix(s, "[") {
		return nil
	}
	var raw map[string]*float64
	if err := json.Unmarshal(b, &raw); err != nil {
		return fmt.Errorf("intermediateNums: %w", err)
	}
	n.m = make(map[string]float64, len(raw))
	for k, v := range raw {
		if v == nil {
			n.nulls++
			continue
		}
		n.m[k] = *v
	}
	return nil
}

type intermediateDiff struct {
	table, rung, key string
	want, got        float64
}

func (d intermediateDiff) bad() bool    { return math.IsNaN(d.got) || math.IsInf(d.got, 0) }
func (d intermediateDiff) missing() int { return b2i(d.bad() || d.got < d.want) }
func (d intermediateDiff) extra() int   { return b2i(d.bad() || d.got > d.want) }
func (d intermediateDiff) dir() string {
	switch {
	case d.bad():
		return "NAN"
	case d.got < d.want:
		return "MISS"
	}
	return "XTRA"
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intermediateFirstDiffs(tables []intermediateTable) ([]intermediateDiff, []string, []string) {
	first := map[string]int{}
	var all []intermediateDiff
	var ranks []int
	unclassified, unwritten := map[string]bool{}, map[string]bool{}
	for _, tb := range tables {
		for k, want := range tb.want {
			got, written := tb.got[k]
			if _, inMain := tb.graded[k]; !inMain && !written {
				unwritten[tb.name+"."+k] = true
				continue
			}
			if !math.IsNaN(got) && !math.IsInf(got, 0) && oracleClose(got, want, oracleDumpTolerance) {
				continue
			}
			r := intermediateRungOf(k)
			if r.chain == intermediateUnclassified {
				unclassified[k] = true
			}
			if f, ok := first[r.chain]; !ok || r.order < f {
				first[r.chain] = r.order
			}
			all = append(all, intermediateDiff{table: tb.name, rung: r.label(), key: k, want: want, got: got})
			ranks = append(ranks, r.order)
		}
	}
	var out []intermediateDiff
	for i, d := range all {
		if ranks[i] == first[strings.SplitN(d.rung, ".", 2)[0]] {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].rung != out[j].rung {
			return out[i].rung < out[j].rung
		}
		if out[i].table != out[j].table {
			return out[i].table < out[j].table
		}
		return out[i].key < out[j].key
	})
	return out, sortedBoolKeys(unclassified), sortedBoolKeys(unwritten)
}

const intermediateUnclassified = "other"

type intermediateRung struct {
	chain, name string
	order       int
}

func (r intermediateRung) label() string { return fmt.Sprintf("%s.%d-%s", r.chain, r.order, r.name) }

const intermediateDT = `(Physical|Lightning|Cold|Fire|Chaos|Elemental)`

var intermediateRungs = []struct {
	re *regexp.Regexp
	r  intermediateRung
}{
	{regexp.MustCompile(`^Self|OnSelf$|AvoidChance$|^Avoid|^DebuffExpiration`), intermediateRung{"defence", "mitigation", 2}},
	{regexp.MustCompile(`EHP|MaximumHitTaken$|TotalPool$|HitPool$|NumberOf(Mitigated)?DamagingHits$|TakenDamage$|TakenHit$|^total(Taken|EnemyDamage)|ManaEffectiveLife$|^preventedLifeLoss|^PvPTotalTakenHit`), intermediateRung{"defence", "ehp", 3}},
	{regexp.MustCompile(`Resist|BlockChance|^Block(Effect|Duration)$|Dodge|SpellSuppression|EvadeChance|NotHitChance|DamageReduction|TakenHitMulti?$|TakenDotMult$|Aegis|GuardAbsorb|MindOverMatter|EnergyShieldBypass|EffectiveAppliedArmour|^` + intermediateDT + `Enemy|^Enemy(Crit|Stun)|Mitigation$|takenFlat$|TakenReflect$|DamageTakenOnBlock|CritExtraDamageReduction|^Stun(Threshold|Duration)$|^(Minimum|Any)Bypass|^specificTypeAvoidance|^Configured|^enemySkillTime$|^noSplitEvade|RecoupableDamageTaken$|^FRDamageTaken`), intermediateRung{"defence", "mitigation", 2}},
	{regexp.MustCompile(`^(With|Combined|Full|Culling|Mirage|Reservation|Pvp)\w*DPS$|^CombinedAvg$|^Total(DPS|PoisonDPS)$|^(Total|` + intermediateDT[1:len(intermediateDT)-1] + `)Dot(DPS|Instance|CalcSection)?$|^AverageBurstDamage$|^impaleStored|^DecayDPS$`), intermediateRung{"offence", "dps", 8}},
	{regexp.MustCompile(`Leech|On(Hit|Kill|Block|SpellBlock|Suppress)(Rate)?$|Regen|Recovery|Recharge|Recoup|Degen|LossLost|Flask|Tincture|^net`), intermediateRung{"recovery", "recovery", 1}},
	{regexp.MustCompile(`Cost(Raw)?$|^SoulGainPrevention`), intermediateRung{"cost", "cost", 1}},
	{regexp.MustCompile(`^(Str|Dex|Int|TotalAttr|LowestAttribute|Devotion)$|^Req(Str|Dex|Int)`), intermediateRung{"setup", "attributes", 1}},
	{regexp.MustCompile(`Charges(Max|Min|Duration)?$|^Removable|Fortif|^(Maximum)?Rage$|^RageEffect|Infusions|GhostShrouds|CrabBarriers|^WarcryPower$`), intermediateRung{"setup", "charges", 2}},
	{regexp.MustCompile(`[Rr]eserv`), intermediateRung{"setup", "reservation", 3}},
	{regexp.MustCompile(`^(Life|Mana|EnergyShield|Ward|Armour|Evasion)$|^Gear:|On(Body Armour|Boots|Gloves|Helmet|Weapon 2)$|^LifeRecoverable$|^LowestOf|^ArmourDefense$|^(Melee|Projectile)Evasion$|^Totem(Life|Armour|EnergyShield|LifeMod)$|Life$|^(Full|Low)LifePercentage$`), intermediateRung{"defence", "pools", 1}},
	{regexp.MustCompile(`Ignite|Bleed|Poison|Chill|Shock|Freeze|Scorch|Brittle|Sap|Impale|DotMulti|Knockback|^Wither|^Blind|^Burning|^Caustic|DotEffMult`), intermediateRung{"offence", "ailment", 7}},
	{regexp.MustCompile(`^` + intermediateDT + `(Min|Max)Base$`), intermediateRung{"offence", "base", 2}},
	{regexp.MustCompile(`^` + intermediateDT + `EffMult$`), intermediateRung{"offence", "effmult", 3}},
	{regexp.MustCompile(`^` + intermediateDT + `(Min|Max)$`), intermediateRung{"offence", "minmax", 4}},
	{regexp.MustCompile(`Crit`), intermediateRung{"offence", "crit", 5}},
	{regexp.MustCompile(`HitAverage$|^AverageHit$|^Total(Min|Max)$|HitChance$|^Accuracy|^Speed$|^Time$|^Hit(Speed|Time)$|^CastRate$|^AverageDamage$|DamageRolls$|^(Double|Triple)Damage|^ScaledDamageEffect$|^Cull|^RuthlessBlow|^allMult$|^Exerted|HitEffect$|AvgDmg$|FistOfWar|NumberOfHits$|^EffectiveSourceRate$|Trigger|Throwing|Laying|Placement|^enemyBlockChance$|^Pvp|U[pP][tT]ime(Ratio)?$|CastTime$|Cooldown$|^TotemActionSpeed$|WarcryEffect$|^(Beidat|Ghorr|Lycia|Pact)|Exerts?Count$|^OverlapChance$|^SkillDPSMultiplier$|^AverageBurstHits$|^ScaledDamage|^Repeats$|^AverageActiveTraps$|^PulsesPerTrap$`), intermediateRung{"offence", "hit", 6}},
	{regexp.MustCompile(`^Gem(Level|Quality)$|Mod$|Radius|^AreaOfEffect|^ProjectileCount$|^Chain|Pierce|^Split|^Repeat|^StrikeTargets$|^Duration|Limit$|^Active|Count$|Range|Metres?$|^SummonedMinionsPerCast$|^QuantityMultiplier$|^StoredUses$|Seals?|^Brand|^AttachedBrand|^CWDTThreshold$|^SmallExplosions|^TotemsSummoned$|^LightRadius|^Loot`), intermediateRung{"offence", "skill", 1}},
}

func intermediateRungOf(key string) intermediateRung {
	for _, x := range intermediateRungs {
		if x.re.MatchString(key) {
			return x.r
		}
	}
	return intermediateRung{intermediateUnclassified, "unclassified", 1}
}

func readWorklistWrong(path string) (map[string]int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("worklist: %w", err)
	}
	defer f.Close()
	out := map[string]int{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for line := 0; sc.Scan(); line++ {
		p := strings.Split(sc.Text(), "\t")
		if line == 0 {
			if len(p) < 2 || p[0] != "build" || p[1] != "wrong" {
				return nil, fmt.Errorf("%s: header %q is not build\\twrong\\t...", path, sc.Text())
			}
			continue
		}
		n, err := strconv.Atoi(p[1])
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, line+1, err)
		}
		out[p[0]] = n
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s: no builds", path)
	}
	return out, nil
}

func sortedCountKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedBoolKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
