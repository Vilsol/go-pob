package mcp

import (
	"cmp"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

// assumptionSections are ConfigOptions.lua's sections whose options assume a combat situation rather than describe the
// character or the enemy.
var assumptionSections = []string{"When In Combat", "For Effective DPS"}

// configSectionOf maps each config var to its ConfigOptions.lua section; a section header entry opens a section.
var configSectionOf = sync.OnceValue(func() map[string]string {
	var all []struct {
		Var     string `json:"var"`
		Section string `json:"section"`
	}
	_ = json.Unmarshal(configOptionsJSON, &all)
	out := map[string]string{}
	section := ""
	for _, o := range all {
		if o.Var == "" {
			if o.Section != "" {
				section = o.Section
			}
			continue
		}
		if _, ok := out[o.Var]; !ok {
			out[o.Var] = section
		}
	}
	return out
})

// stageOptions are ConfigOptions.lua's skill options that set a stage or stack count: set, they assume the skill holds
// that count all the time.
var stageOptions = sync.OnceValue(func() map[string]bool {
	out := map[string]bool{}
	for _, o := range configOptions() {
		l := strings.ToLower(o.Var + " " + o.Label)
		if (o.Type == "count" || o.Type == "countAllowZero") && configSectionOf()[o.Var] == "Skill Options" && (strings.Contains(l, "stage") || strings.Contains(l, "stack")) {
			out[o.Var] = true
		}
	}
	return out
})

// forcedConditions lists the build's config inputs that assume a combat situation, or a skill's stage or stack count,
// and are set away from their default.
func forcedConditions(b *pob.PathOfBuilding) []string {
	defaults := map[string]any{}
	for _, o := range configOptions() {
		defaults[o.Var] = o.defaultValue()
	}
	var out []string
	for _, in := range b.Config.Inputs {
		if (!slices.Contains(assumptionSections, configSectionOf()[in.Name]) && !stageOptions()[in.Name]) || slices.Contains(out, in.Name) {
			continue
		}
		var v any
		switch {
		case in.Boolean != nil:
			if !*in.Boolean {
				continue
			}
			v = true
		case in.Number != nil:
			if *in.Number == 0 {
				continue
			}
			v = *in.Number
		case in.String != nil:
			v = *in.String
		default:
			continue
		}
		if d, ok := defaults[in.Name]; ok && fmt.Sprint(d) == fmt.Sprint(v) {
			continue
		}
		out = append(out, in.Name)
	}
	return out
}

var ailmentChances = []string{"BleedChance", "PoisonChance", "IgniteChance", "ShockChance", "FreezeChance", "ChillChance", "ScorchChance", "BrittleChance", "SapChance"}

// lowestMaxHit is the smallest of the per-type maximum hits PoB shows.
func lowestMaxHit(env *calculator.Environment) float64 {
	out := math.Inf(1)
	for _, t := range []string{"Physical", "Fire", "Cold", "Lightning", "Chaos"} {
		if v, ok := env.Player.Output[t+"MaximumHitTaken"]; ok {
			out = math.Min(out, v)
		}
	}
	if math.IsInf(out, 1) {
		return 0
	}
	return out
}

// dpsColumn is the stat dDPS shows: the ranked stat when it is a DPS, else main DPS.
func dpsColumn(key string) string {
	if strings.Contains(key, "DPS") {
		return key
	}
	return mainDPS
}

// changeColumns are the default per-row differences: dpsKey's DPS, FullDPS on a build with Full DPS skills (MainDPS
// when dpsKey is FullDPS), EHP, lowest max hit, life, totem life and unreserved mana; a zero difference is left out.
func changeColumns(base, after *calculator.Environment, dpsKey string) map[string]float64 {
	out := map[string]float64{}
	put := func(name string, a, b float64) {
		if d := round(b - a); d != 0 {
			out[name] = d
		}
	}
	bd, _ := statValue(base, dpsKey)
	ad, _ := statValue(after, dpsKey)
	put("dDPS", bd, ad)
	if bf := base.Player.Output["FullDPS"]; dpsKey != "FullDPS" && bf > 0 {
		put("dFullDPS", bf, after.Player.Output["FullDPS"])
	}
	if dpsKey == "FullDPS" {
		bm, _ := statValue(base, mainDPS)
		am, _ := statValue(after, mainDPS)
		put("dMainDPS", bm, am)
	}
	put("dEHP", base.Player.Output["TotalEHP"], after.Player.Output["TotalEHP"])
	put("dMaxHit", lowestMaxHit(base), lowestMaxHit(after))
	put("dLife", base.Player.Output["Life"], after.Player.Output["Life"])
	if _, ok := base.Player.Output["TotemLife"]; ok {
		put("dTotemLife", base.Player.Output["TotemLife"], after.Player.Output["TotemLife"])
	}
	if _, ok := base.Player.Output["ManaUnreserved"]; ok {
		put("dManaUnreserved", base.Player.Output["ManaUnreserved"], after.Player.Output["ManaUnreserved"])
	}
	return out
}

// socketOverflow lists the slots whose groups socket more enabled gems than the slot's item has sockets.
func socketOverflow(b *pob.PathOfBuilding) map[string][2]int {
	out := map[string][2]int{}
	seen := map[string]bool{}
	for _, g := range *groupsOf(b) {
		if g.Slot == "" || seen[g.Slot] || g.Source != "" || !g.Enabled {
			continue
		}
		seen[g.Slot] = true
		if sockets, gemsIn := slotSockets(b, g.Slot); gemsIn > sockets {
			out[g.Slot] = [2]int{gemsIn, sockets}
		}
	}
	return out
}

// realismFlags names what a change does that its stat gain does not show: new build warnings, overfilled sockets, a
// two-handed weapon that drops the off-hand, skills it disables, a lower totem limit or more totems summoned than the limit, and lost immunities or ailment chances.
func realismFlags(b, after *pob.PathOfBuilding, baseEnv, afterEnv *calculator.Environment) []string {
	var flags []string
	baseWarn := warningsOf(baseEnv)
	for _, w := range warningsOf(afterEnv) {
		if !slices.Contains(baseWarn, w) {
			if len(w) > 80 {
				w = w[:77] + "..."
			}
			flags = append(flags, "warning: "+w)
		}
	}
	before := socketOverflow(b)
	for slot, o := range socketOverflow(after) {
		if before[slot] != o {
			flags = append(flags, fmt.Sprintf("sockets: %d gems in %d sockets of %s", o[0], o[1], slot))
		}
	}
	if afterEnv.Player.ModDB.Conditions["UsingTwoHandedWeapon"] && !baseEnv.Player.ModDB.Conditions["UsingTwoHandedWeapon"] {
		if w := slotItemName(b, "Weapon 2"); w != "" {
			flags = append(flags, "removes Weapon 2: "+w)
		}
	}
	for _, d := range disabledSkills(baseEnv, afterEnv) {
		flags = append(flags, "disables: "+d)
	}
	bo, ao := baseEnv.Player.Output, afterEnv.Player.Output
	if l, ok := bo["ActiveTotemLimit"]; ok && ao["ActiveTotemLimit"] < l {
		flags = append(flags, fmt.Sprintf("totems: limit %v -> %v", l, ao["ActiveTotemLimit"]))
	}
	if s, l := ao["TotemsSummoned"], ao["ActiveTotemLimit"]; s > l && bo["TotemsSummoned"] <= bo["ActiveTotemLimit"] {
		flags = append(flags, fmt.Sprintf("totems: %v summoned, limit %v", s, l))
	}
	return append(flags, lossFlags(baseEnv, afterEnv)...)
}

// lossFlags names the capped avoidances a change loses (an ailment's 100%, spell suppression's 100%, CalcDefence.lua:1146,
// 1620-1624), Corrupted Blood immunity (CalcDefence.lua:1590), the ailment chances it takes to 0 and the shock it lowers.
func lossFlags(baseEnv, afterEnv *calculator.Environment) []string {
	bo, ao := baseEnv.Player.Output, afterEnv.Player.Output
	var lost []string
	for k, v := range bo {
		if (strings.HasSuffix(k, "AvoidChance") || k == "SpellSuppressionChance") && v >= 100 && ao[k] < 100 {
			lost = append(lost, fmt.Sprintf("%s %v -> %v", k, round(v), round(ao[k])))
		}
	}
	if baseEnv.Player.ModDB.Flag(nil, "CorruptedBloodImmune") && !afterEnv.Player.ModDB.Flag(nil, "CorruptedBloodImmune") {
		lost = append(lost, "CorruptedBloodImmunity")
	}
	for _, k := range ailmentChances {
		if bo[k] > 0 && ao[k] <= 0 {
			lost = append(lost, fmt.Sprintf("%s %v -> 0", k, round(bo[k])))
		}
	}
	if v, ok := bo["CurrentShock"]; ok && ao["CurrentShock"] < v {
		lost = append(lost, fmt.Sprintf("CurrentShock %v -> %v", round(v), round(ao["CurrentShock"])))
	}
	slices.Sort(lost)
	out := make([]string, 0, len(lost))
	for _, l := range lost {
		out = append(out, "loses: "+l)
	}
	return out
}

// slowerFlag names a lower main-skill attack or cast rate, which no DPS column shows: with a DPS gain, PoB's number, but
// one that rises as the skill slows (an exert's uptime divides by speed) is worth a second look.
func slowerFlag(base, after *calculator.Environment, dpsKey string) []string {
	bs, as := base.Player.Output["Speed"], after.Player.Output["Speed"]
	bd, _ := statValue(base, dpsKey)
	ad, _ := statValue(after, dpsKey)
	if as <= 0 || as >= bs*0.995 {
		return nil
	}
	way := "rises"
	if ad < bd {
		way = "falls"
	} else if ad == bd {
		way = "holds"
	}
	sp := func(v float64) float64 { return math.Round(v*100) / 100 }
	return []string{fmt.Sprintf("slower: Speed %v -> %v while %s %s", sp(bs), sp(as), dpsKey, way)}
}

// fullDPSFlag names a change that moves dpsKey and FullDPS in opposite directions, on a build with Full DPS skills.
func fullDPSFlag(base, after *calculator.Environment, dpsKey string) []string {
	bf := base.Player.Output["FullDPS"]
	if dpsKey == "FullDPS" || bf <= 0 {
		return nil
	}
	bd, _ := statValue(base, dpsKey)
	ad, _ := statValue(after, dpsKey)
	af := after.Player.Output["FullDPS"]
	dp, fp := pct(bd, ad), pct(bf, af)
	if dp == nil || fp == nil || *dp == 0 || *fp == 0 || (*dp > 0) == (*fp > 0) {
		return nil
	}
	return []string{fmt.Sprintf("full dps: FullDPS %+g%% while %s %+g%%", *fp, dpsKey, *dp)}
}

// realism re-checks ranked rows: flags, default change columns and, against the build with its favourable combat
// conditions questioned, which of those conditions a row's value depends on.
type realism struct {
	b      *pob.PathOfBuilding
	key    string
	base   *calculator.Environment
	forced []string
	// off are the unset conditions a modifier reads negated; questioning turns them on
	off      []string
	sources  map[string]bool
	worlds   map[string]*world
	offEnvs  map[string]*calculator.Environment
	deadline time.Time
	// overLimit marks a base whose Full DPS counts exceed their totem or mine limit; scored that a row got a worst score
	overLimit bool
	scored    bool
	clamped   *calculator.Environment
	// bases keeps on each worst-scored row the stat and TotalEHP of the calculation its worst scores start from
	bases bool
	// value reads a row's stat from a calculation, counting the totems it no longer summons
	value func(base, env *calculator.Environment) float64
}

// world is the build with the forced conditions off and a set of assumed-off conditions on.
type world struct {
	b       *pob.PathOfBuilding
	env     *calculator.Environment
	clamped *calculator.Environment
}

func newRealism(b *pob.PathOfBuilding, env *calculator.Environment, key string, deadline time.Time) *realism {
	r := &realism{b: b, key: key, base: env, forced: forcedConditions(b), worlds: map[string]*world{}, offEnvs: map[string]*calculator.Environment{}, deadline: deadline}
	r.value = func(base, env *calculator.Environment) float64 { return statOf(env, key) * totemShare(key, base, env) }
	r.off = assumedOff(b, env)
	if len(r.forced) > 0 {
		r.sources = sourceLines(b)
	}
	r.overLimit = len(countsOverLimit(env)) > 0
	return r
}

func (r *realism) worldFor(off []string) *world {
	k := strings.Join(off, ",")
	w, ok := r.worlds[k]
	if !ok {
		w = &world{b: questioned(r.b, r.forced, off)}
		w.env = calcEnv(w.b)
		r.worlds[k] = w
	}
	return w
}

// clampedBase is the calculation worst scores start from: w's, or the build's without one, with Full DPS counts held
// to their limits.
func (r *realism) clampedBase(w *world) *calculator.Environment {
	if w == nil {
		if r.clamped == nil {
			r.clamped = clampedEnv(r.b, r.base)
		}
		return r.clamped
	}
	if w.clamped == nil {
		w.clamped = clampedEnv(w.b, w.env)
	}
	return w.clamped
}

func (r *realism) gain(baseEnv, env *calculator.Environment) float64 {
	return r.value(baseEnv, env) - r.value(baseEnv, baseEnv)
}

// share is a gain as a fraction of the stat on baseEnv; the gain itself when the stat is 0 there.
func (r *realism) share(baseEnv *calculator.Environment, g float64) float64 {
	if v := math.Abs(r.value(baseEnv, baseEnv)); v > 0 {
		return g / v
	}
	return g
}

// differs reports whether two relative gains of one change disagree by more than half the first, or by 1 point of the
// stat; gains scale with the build, so a condition that only rescales them is not one they depend on.
func differs(a, b float64) bool {
	return math.Abs(a-b) > math.Max(0.5*math.Abs(a), 0.01)
}

// rankedColumn is the change column that shows the ranked stat key; empty when none does.
func rankedColumn(key string) string {
	if dpsColumn(key) == key {
		return "dDPS"
	}
	return map[string]string{"TotalEHP": "dEHP", "Life": "dLife", "TotemLife": "dTotemLife", "ManaUnreserved": "dManaUnreserved"}[key]
}

// changeKey holds a row's change until annotate takes it off.
const changeKey = "\x00change"

// worstCapNote ends every worst_case note: why a worst_gain can sit at the gain's share.
const worstCapNote = "; a worst_gain is capped at the gain's share of the build where the change itself provides a questioned condition, and worst_d's column of the ranked stat with it"

// worstBaseKey holds, on a worst-scored row of a realism with bases, the ranked stat ("stat") and TotalEHP of the
// calculation its worst scores start from.
const worstBaseKey = "\x00worstbase"

const (
	// bigLossShare is the share of a watched stat's value a change may take off before big_loss names it
	bigLossShare = 0.2
	// bigResistDrop is the resistance points a change may take off before big_loss names it
	bigResistDrop = 20.0
)

// bigLosses names the watched stats other than the ranked key that cols (after against base) take down by
// bigLossShare of their value on base or more, and the resistances after lowers by bigResistDrop points or more.
// MainDPS and FullDPS watch each other: ranking one, the other is watched.
func bigLosses(key string, base, after *calculator.Environment, cols map[string]float64) []string {
	dpsKey := dpsColumn(key)
	bd, _ := statValue(base, dpsKey)
	bm, _ := statValue(base, mainDPS)
	bo := base.Player.Output
	watched := []struct {
		col, name string
		was       float64
		skip      bool
	}{
		{"dDPS", dpsKey, bd, dpsKey == key},
		{"dFullDPS", "FullDPS", bo["FullDPS"], key == "FullDPS"},
		{"dMainDPS", mainDPS, bm, key == mainDPS},
		{"dEHP", "TotalEHP", bo["TotalEHP"], key == "TotalEHP"},
		{"dMaxHit", "lowest MaximumHitTaken", lowestMaxHit(base), strings.HasSuffix(key, "MaximumHitTaken")},
		{"dLife", "Life", bo["Life"], key == "Life"},
		{"dTotemLife", "TotemLife", bo["TotemLife"], key == "TotemLife"},
	}
	var out []string
	for _, w := range watched {
		if d := cols[w.col]; !w.skip && w.was > 0 && d < -bigLossShare*w.was {
			out = append(out, fmt.Sprintf("%s %v%%", w.name, round(d/w.was*100)))
		}
	}
	for _, el := range []string{"Fire", "Cold", "Lightning", "Chaos"} {
		k := el + "Resist"
		if k == key {
			continue
		}
		if a, b := bo[k], after.Player.Output[k]; a-b >= bigResistDrop {
			out = append(out, fmt.Sprintf("%s %v -> %v", k, round(a), round(b)))
		}
	}
	return out
}

// annotate adds flags and change columns to rows and takes their changes off; with worst, also the gain with the
// favourable conditions questioned as worst_gain (worst_loss when sign is -1, for rows that rank a loss) and the change
// columns there as worst_d. A worst_gain is not a larger share of its base than the gain is of the build's (as differs
// tells shares apart): where the change itself provides a questioned condition it takes the gain's share, as
// worstCapNote tells the caller. A row also questions the assumed-off conditions its own change starts reading. It reports
// whether it ran out of time.
func (r *realism) annotate(rows []map[string]any, sign float64, worst bool) bool {
	cut := false
	for _, row := range rows {
		ch, ok := row[changeKey].(change)
		delete(row, changeKey)
		if !ok || cut {
			continue
		}
		if time.Now().After(r.deadline) {
			cut = true
			continue
		}
		c := cloneBuild(r.b)
		ch(c)
		after := calcEnv(c)
		flags := realismFlags(r.b, c, r.base, after)
		if r.sources != nil {
			flags = append(flags, sourceFlags(r.forced, r.sources, sourceLines(c))...)
		}
		flags = append(flags, slowerFlag(r.base, after, dpsColumn(r.key))...)
		flags = append(flags, areaFlag(r.base, after, dpsColumn(r.key))...)
		counted, countFlags := after, []string(nil)
		if dpsColumn(r.key) != "FullDPS" {
			counted, countFlags = countsAfter(c, r.base, after)
		}
		flags = append(flags, fullDPSFlag(r.base, counted, dpsColumn(r.key))...)
		flags = append(flags, customModsFlags(r.b.Build.PassiveNodes, c)...)
		flags = append(flags, countFlags...)
		cols := changeColumns(r.base, after, dpsColumn(r.key))
		if counted != after {
			delete(cols, "dFullDPS")
			if d := round(counted.Player.Output["FullDPS"] - r.base.Player.Output["FullDPS"]); d != 0 {
				cols["dFullDPS"] = d
			}
		}
		totemFlag := r.perTotem(cols, r.base, after)
		if len(cols) > 0 {
			row["d"] = cols
		}
		off := slices.Clone(r.off)
		for _, o := range assumedOff(c, after) {
			if !slices.Contains(off, o) {
				off = append(off, o)
			}
		}
		var w *world
		worstBuilt, worstAfter := c, after
		if len(r.forced)+len(off) > 0 {
			w = r.worldFor(off)
			g := r.gain(r.base, after)
			worstBuilt = cloneBuild(w.b)
			ch(worstBuilt)
			worstAfter = calcEnv(worstBuilt)
			if differs(r.share(r.base, g), r.share(w.env, r.gain(w.env, worstAfter))) {
				if names := r.blame(ch, g, off); len(names) > 0 {
					flags = append(flags, "config: "+strings.Join(names, ","))
				} else {
					flags = append(flags, "config: only the questioned conditions together")
				}
			}
		}
		basisBase, basisAfter, basisCols := r.base, after, cols
		uptime, timed := flaskUptime(r.b, c)
		if worst && (w != nil || r.overLimit || len(countsOverLimit(after)) > 0 || len(timed) > 0) {
			base := r.clampedBase(w)
			wa := clampedEnv(worstBuilt, worstAfter)
			g := r.gain(base, wa)
			wd := changeColumns(base, wa, dpsColumn(r.key))
			if f := r.perTotem(wd, base, wa); totemFlag == "" {
				totemFlag = f
			}
			if len(timed) > 0 {
				offBuilt := cloneBuild(worstBuilt)
				for _, sl := range timed {
					offBuilt.SetFlaskActive(sl, false)
				}
				oa := clampedEnv(offBuilt, calcEnv(offBuilt))
				g = uptime*g + (1-uptime)*r.gain(base, oa)
				wd = blendColumns(wd, changeColumns(base, oa, dpsColumn(r.key)), uptime)
				flags = append(flags, fmt.Sprintf("uptime: worst_gain counts %s on %g%% of the time", strings.Join(timed, ","), math.Round(uptime*100)))
			}
			if g0, wv := r.gain(r.base, after), math.Abs(r.value(base, base)); sign > 0 && wv > 0 && r.value(r.base, r.base) != 0 && g/wv > r.share(r.base, g0) && differs(r.share(r.base, g0), g/wv) {
				g = r.share(r.base, g0) * wv
				// worst_d's column of the ranked stat says what worst_gain counts
				if col := rankedColumn(r.key); col != "" {
					delete(wd, col)
					if d := round(g); d != 0 {
						wd[col] = d
					}
				}
			}
			row["worst_"+map[bool]string{true: "gain", false: "loss"}[sign > 0]] = round(sign*g) + 0
			row["worst_d"] = wd
			if r.bases {
				row[worstBaseKey] = map[string]float64{"stat": r.value(base, base), "TotalEHP": base.Player.Output["TotalEHP"]}
			}
			basisBase, basisAfter, basisCols = base, wa, wd
			r.scored = true
		}
		if bl := bigLosses(r.key, basisBase, basisAfter, basisCols); len(bl) > 0 {
			row["big_loss"] = bl
		}
		if totemFlag != "" {
			flags = append(flags, totemFlag)
		}
		if own, _ := row["flags"].([]string); len(own) > 0 {
			flags = append(own, flags...)
		}
		if len(flags) > 0 {
			row["flags"] = flags
		}
	}
	return cut
}

// perTotem sets cols' dDPS to the change the row's gain counts when the ranked stat is a totem skill's DPS, which is per
// totem, and after summons fewer totems than base; it returns the flag that says so.
func (r *realism) perTotem(cols map[string]float64, base, after *calculator.Environment) string {
	if dpsColumn(r.key) != r.key || totemShare(r.key, base, after) == 1 {
		return ""
	}
	delete(cols, "dDPS")
	if d := round(r.gain(base, after)); d != 0 {
		cols["dDPS"] = d
	}
	return fmt.Sprintf("totems: %v -> %v summoned; gain and dDPS count the totems lost, as %s is per totem", base.Player.Output["TotemsSummoned"], after.Player.Output["TotemsSummoned"], r.key)
}

// blame names the questioned conditions whose toggle alone moves the change's relative gain by a tenth or more, the
// largest first and at most four; none when only their combination does.
func (r *realism) blame(ch change, g float64, off []string) []string {
	type mover struct {
		name string
		by   float64
	}
	var movers []mover
	share := r.share(r.base, g)
	for _, f := range slices.Concat(r.forced, off) {
		if time.Now().After(r.deadline) {
			break
		}
		forced, off := []string{f}, []string(nil)
		label := f
		if !slices.Contains(r.forced, f) {
			forced, off, label = nil, []string{f}, f+"=on"
		}
		tb := questioned(r.b, forced, off)
		env, ok := r.offEnvs[f]
		if !ok {
			env = calcEnv(tb)
			r.offEnvs[f] = env
		}
		by := math.Abs(share - r.share(env, r.gain(env, evalChange(tb, ch))))
		if by > math.Max(0.1*math.Abs(share), 0.0005) {
			movers = append(movers, mover{label, by})
		}
	}
	slices.SortStableFunc(movers, func(a, b mover) int { return cmp.Compare(b.by, a.by) })
	out := make([]string, 0, min(len(movers), 4))
	for _, m := range movers[:min(len(movers), 4)] {
		out = append(out, m.name)
	}
	return out
}

var (
	customWordRe = regexp.MustCompile(`[a-z]{4,}`)
	// words too common in mod lines to tie a custom modifier to a passive line
	customCommonWords = map[string]bool{"chance": true, "increased": true, "reduced": true, "more": true, "less": true,
		"damage": true, "with": true, "from": true, "your": true, "have": true, "that": true, "while": true, "when": true,
		"enemies": true, "enemy": true, "critical": true, "strike": true, "attacks": true, "attack": true, "skills": true,
		"skill": true, "seconds": true, "second": true, "lasting": true, "maximum": true, "additional": true}
)

func customWords(line string) []string {
	var out []string
	for _, w := range customWordRe.FindAllString(strings.ToLower(line), -1) {
		if !customCommonWords[w] && !slices.Contains(out, w) {
			out = append(out, w)
		}
	}
	return out
}

// customStandIn is whether an enabled custom modifier line could stand in for an unparsed passive line: it shares two
// of the line's distinctive words, or its only one.
func customStandIn(custom []string, line string) bool {
	words := customWords(line)
	need := min(2, len(words))
	return need > 0 && slices.ContainsFunc(custom, func(c string) bool {
		n := 0
		for _, w := range customWords(c) {
			if slices.Contains(words, w) {
				n++
			}
		}
		return n >= need
	})
}

// customModsFlags names the passive lines PoB cannot parse that a change removes from baseNodes, when an enabled custom
// modifier line shares the line's distinctive words: it may stand in for the line and still applies, so the change
// shows no loss.
func customModsFlags(baseNodes []int64, after *pob.PathOfBuilding) []string {
	var custom []string
	for _, c := range after.GetCustomModifierBlocks() {
		if c.IsEnabled() {
			for _, l := range strings.Split(c.Text, "\n") {
				if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "#") {
					custom = append(custom, l)
				}
			}
		}
	}
	if len(custom) == 0 {
		return nil
	}
	v := specTreeVersion(after)
	var out []string
	for _, id := range baseNodes {
		if slices.Contains(after.Build.PassiveNodes, id) {
			continue
		}
		n, ok := treeNodeByID(v, id)
		if !ok {
			continue
		}
		lines := slices.DeleteFunc(unparsedNodeLines(n.Stats), func(l string) bool { return !customStandIn(custom, l) })
		if len(lines) > 0 {
			l := lines[0]
			if len(l) > 60 {
				l = l[:57] + "..."
			}
			out = append(out, fmt.Sprintf("customMods: removes %s's %q, which PoB cannot parse; a custom modifier standing in for it still applies", n.Name, l))
		}
	}
	return out
}

// areaFlag names a DPS gain that comes with a smaller main-skill area of effect (Concentrated Effect's trade).
func areaFlag(base, after *calculator.Environment, dpsKey string) []string {
	bm, am := base.Player.Output["AreaOfEffectMod"], after.Player.Output["AreaOfEffectMod"]
	bd, _ := statValue(base, dpsKey)
	ad, _ := statValue(after, dpsKey)
	if ad <= bd || am <= 0 || am >= bm*0.995 {
		return nil
	}
	sp := func(v float64) float64 { return math.Round(v*100) / 100 }
	f := fmt.Sprintf("area: AreaOfEffectMod %v -> %v", sp(bm), sp(am))
	if br, ar := base.Player.Output["AreaOfEffectRadius"], after.Player.Output["AreaOfEffectRadius"]; br > 0 && ar > 0 {
		f += fmt.Sprintf(" (radius %v -> %v)", br, ar)
	}
	return []string{f + " while " + dpsKey + " rises"}
}

// flaskUptimeOf is the share of the time a utility flask's effect can be up: PoB counts every active flask as on all
// the time, which holds for a flask whose charge income keeps its base up; at that income a flask is up for its effect
// seconds per charge used over its base's (Item.lua:2429-2450), at most all the time. A life or mana flask's duration is
// its recovery, not an effect, and is always 1.
func flaskUptimeOf(item *calculator.Item) float64 {
	if item.Base == nil || item.Base.Flask == nil {
		return 1
	}
	fb, fd := item.Base.Flask, item.FlaskData
	if fb.Life > 0 || fb.Mana > 0 || fb.Duration <= 0 || fb.ChargesUsed <= 0 || fd["duration"] <= 0 || fd["chargesUsed"] <= 0 {
		return 1
	}
	return min(1, fd["duration"]/fd["chargesUsed"]/(fb.Duration/float64(fb.ChargesUsed)))
}

// flaskUptime is the lowest uptime among the flasks after switches on that b did not have on, and the slots of those
// up less than all the time.
func flaskUptime(b, after *pob.PathOfBuilding) (float64, []string) {
	was := map[string]equipped{}
	for _, e := range equippedItems(b) {
		was[e.slot] = e
	}
	uptime := 1.0
	var timed []string
	for _, e := range equippedItems(after) {
		if !strings.HasPrefix(e.slot, "Flask") || !e.active || (was[e.slot].active && was[e.slot].text == e.text) {
			continue
		}
		if u := flaskUptimeOf(calculator.ParseItem(e.text)); u < 1 {
			uptime = min(uptime, u)
			timed = append(timed, e.slot)
		}
	}
	return uptime, timed
}

// blendColumns is the change columns of a flask up share of the time: on's at that share, off's for the rest.
func blendColumns(on, off map[string]float64, share float64) map[string]float64 {
	out := map[string]float64{}
	for _, m := range []map[string]float64{on, off} {
		for k := range m {
			if d := round(share*on[k] + (1-share)*off[k]); d != 0 {
				out[k] = d
			}
		}
	}
	return out
}
