package mcp

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

const (
	scanBudget = 50 * time.Second
	capScan    = 10 * 1024
)

var scanParts = []string{"uniques", "flasks", "affixes", "anoints", "clusters", "reservation", "resists", "sockets", "links", "gems", "swaps"}

// defaultScanParts start with gems, the quickest, so a slow build still sees its gem rows.
var defaultScanParts = []string{"gems", "uniques", "flasks", "affixes", "anoints"}

// savingsScanParts are the default parts under rank_by savings: those whose rows can save chaos.
var savingsScanParts = []string{"uniques", "flasks", "links"}

var gearSlots = []string{"Weapon 1", "Weapon 2", "Helmet", "Body Armour", "Gloves", "Boots", "Amulet", "Ring 1", "Ring 2", "Belt"}

var flaskSlots = []string{"Flask 1", "Flask 2", "Flask 3", "Flask 4", "Flask 5"}

type scanIn struct {
	Handle      string   `json:"handle"`
	Stat        string   `json:"stat"`
	Parts       []string `json:"parts"`
	Slots       []string `json:"slots"`
	PerSlot     int      `json:"per_slot"`
	KeepResists bool     `json:"keep_resists"`
	MaxLevelReq *int     `json:"max_level_req"`
	ManaNeeded  float64  `json:"mana_needed"`
	MaxPoints   *int     `json:"max_points"`
	Apply       string   `json:"apply"`
	DryRun      bool     `json:"dry_run"`
	Watch       []string `json:"watch"`
	Roll        string   `json:"roll"`
	Tier        int      `json:"tier"`
	CapChaos    bool     `json:"cap_chaos"`
	Delve       bool     `json:"delve"`
	Downgrades  bool     `json:"downgrades"`
	budgetIn
}

var scanRolls = map[string]float64{"min": 0, "avg": 0.5, "max": 1}

// scanCtx is one scan's shared state: the build, its misc calculator and the stat every part ranks by.
type scanCtx struct {
	b       *pob.PathOfBuilding
	env     *calculator.Environment
	misc    *calculator.MiscCalculator
	key     string
	ps      calculator.PowerStat
	in      scanIn
	base    float64
	room    map[string]float64
	roll    affixRoll
	partial bool
	cb      *costBudget
	// per part: candidates scored, those that gain, those keep_resists dropped and those cost_budget dropped
	seen, gaining, gapped, priced int
}

func (s *Server) scanBuild(args json.RawMessage) (any, *ToolError) {
	var in scanIn
	if e := decode(args, schemaScanBuild, &in); e != nil {
		return nil, e
	}
	if in.Apply != "" {
		return s.scanApply(in)
	}
	h, e := s.get(in.Handle)
	if e != nil {
		return nil, e
	}
	b, e := h.parseRev(h.cur)
	if e != nil {
		return nil, e
	}
	env, e := h.envAt(h.cur)
	if e != nil {
		return nil, e
	}
	key := in.Stat
	if key == "" {
		key = mainDPS
	}
	ps, e := powerStat(env, key)
	if e != nil {
		return nil, e
	}
	parts := in.Parts
	if len(parts) == 0 {
		parts = defaultScanParts
		if in.RankBy == rankBySavings || in.FitBudget {
			parts = savingsScanParts
		}
	}
	if ps.Combined && !slices.Equal(parts, []string{"flasks"}) {
		return nil, invalidArg("scan_build ranks OffenceDefence in the flasks part only (parts:[\"flasks\"]); rank_candidates ranks the rest by it.", nil)
	}
	for _, p := range parts {
		if !slices.Contains(scanParts, p) {
			return nil, invalidArg(fmt.Sprintf("Unknown part %q.", p), scanParts)
		}
	}
	if slices.Contains(parts, "reservation") && in.ManaNeeded < 0 {
		return nil, invalidArg("mana_needed must be positive.", nil)
	}
	slots := gearSlots
	if len(in.Slots) > 0 {
		slots = nil
		for _, sl := range in.Slots {
			r, e := resolveSlot(sl)
			if e != nil {
				return nil, e
			}
			slots = append(slots, r)
		}
	}
	if in.PerSlot <= 0 {
		in.PerSlot = 2
	}
	in.PerSlot = min(in.PerSlot, 5)
	roll := defaultRoll
	if in.Roll != "" {
		r, ok := scanRolls[in.Roll]
		if !ok {
			return nil, invalidArg(fmt.Sprintf("Unknown roll %q.", in.Roll), []string{"min", "avg", "max"})
		}
		roll.rng = r
	}
	if in.Tier < 0 {
		return nil, invalidArg("tier counts from 1, the best tier.", nil)
	}
	roll.tier = in.Tier
	roll.delve = in.Delve
	cb, e := s.newCostBudget(in.budgetIn, b, false)
	if e != nil {
		return nil, e
	}
	start := time.Now()
	misc := calculator.GetMiscCalculator(b)
	sc := &scanCtx{b: b, env: env, misc: misc, key: key, ps: ps, in: in, base: statOf(misc.Base(), key), room: headroom(misc.Base()), roll: roll, cb: cb}
	res := map[string]any{"handle": h.id, "revision": h.cur, "stat": key, "base": round(sc.base), "base_headroom": sc.room}
	var all []map[string]any
	empty := map[string]string{}
	for i, p := range parts {
		sc.seen, sc.gaining, sc.gapped, sc.priced = 0, 0, 0, 0
		wasPartial := sc.partial
		sc.partial = false
		deadline := start.Add(time.Duration(float64(scanBudget) * float64(i+1) / float64(len(parts)+1)))
		var rows []map[string]any
		switch p {
		case "uniques":
			rows = sc.uniques(slots)
		case "flasks":
			rows = sc.flasks(deadline)
		case "affixes":
			var open []any
			afSlots := slots
			if len(in.Slots) == 0 {
				afSlots = slices.Concat(gearSlots, flaskSlots)
			}
			rows, open = sc.affixes(afSlots, deadline)
			if len(open) > 0 {
				res["open_affixes"] = open
			}
		case "anoints":
			var unmodelled []string
			rows, unmodelled = sc.anoints(deadline)
			if len(unmodelled) > 0 {
				res["anoints_unmodelled"] = map[string]any{"count": len(unmodelled), "examples": unmodelled[:min(5, len(unmodelled))]}
			}
		case "clusters":
			rows = sc.clusters(deadline)
		case "reservation":
			var info map[string]any
			rows, info = sc.reservation(deadline)
			res["reservation"] = info
		case "resists":
			res["resist_plan"] = sc.resists()
		case "sockets":
			rows = sc.sockets(slots, deadline)
		case "links":
			rows = sc.links(slots, deadline)
		case "gems":
			rows = sc.gems(deadline)
		case "swaps":
			rows = sc.swaps(deadline)
		}
		for _, r := range rows {
			r["part"] = p
		}
		if cb != nil {
			rows = slices.DeleteFunc(rows, func(r map[string]any) bool {
				if _, ok := r["cost"]; ok {
					return false
				}
				ch, _ := r[changeKey].(change)
				return ch != nil && !cb.admitChange(r, b, ch)
			})
		}
		if len(rows) == 0 && p != "resists" {
			empty[p] = sc.emptyReason(p)
		}
		sc.partial = sc.partial || wasPartial
		all = append(all, rows...)
	}
	var items []resItem
	for _, r := range all {
		if r["resist_gap"] != nil {
			if items == nil {
				items = resistItems(b, roll.rng)
			}
			r["refill"] = refill(items, sc.room, r, roll)
		}
	}
	realKey := key
	if ps.Combined {
		realKey = mainDPS
	}
	rz := newRealism(b, env, realKey, start.Add(scanBudget))
	rz.bases = true
	if rz.annotate(all, 1, !ps.Combined) {
		sc.partial = true
	}
	if ps.Combined {
		ehpNote(all, env)
	}
	for _, r := range all {
		// a reservation row gains unreserved mana, which the questioned conditions do not hold up; its worst case is
		// the DPS one and would rank it by damage
		if r["part"] == "reservation" {
			delete(r, "worst_gain")
		}
		// a flask row's unmodelled flags name in full the lines its unsupported-mod warnings cut short
		if flags, _ := r["flags"].([]string); r["part"] == "flasks" && slices.ContainsFunc(flags, func(f string) bool { return strings.HasPrefix(f, "unmodelled: ") }) {
			own := fmt.Sprintf("warning: Unsupported mod in %s (%s)", r["slot"], r["name"])
			r["flags"] = slices.DeleteFunc(flags, func(f string) bool { return strings.HasPrefix(f, own) })
		}
	}
	savings := cb != nil && cb.rankBy == rankBySavings
	markUnusable(all)
	markBigLoss(all)
	switch {
	case rz.scored && savings:
		res["worst_case"] = "loss_pct counts worst_gain where a row has one: go-pob's estimate, not PoB's, with forced_conditions off, assumed_off_conditions on and Full DPS counts held to their totem or mine limit" + worstCapNote
	case rz.scored:
		all = rankWorst(all, empty)
		res["worst_case"] = "results rank by worst_gain where a row has one, else gain, unusable rows and rows with big_loss after the clean ones: go-pob's estimate, not PoB's, with forced_conditions off, assumed_off_conditions on and Full DPS counts held to their totem or mine limit; rows that gain nothing there are dropped" + worstCapNote
	default:
		sortUnusableLast(all)
	}
	if cb != nil {
		if savings {
			cb.rankSavings(all, sc.base, env.Player.Output["TotalEHP"], key == "TotalEHP")
			if cb.fit {
				res["fit_budget"] = sc.fitPlan(all, rz)
			}
		} else {
			for _, r := range all {
				score, of := rowScore(r)
				cb.putPerDiv(r, score, of, sc.base, env.Player.Output["TotalEHP"], key == "TotalEHP")
			}
			if cb.rankBy == rankByGainPerDiv {
				sortPerDiv(all)
			}
		}
		res["cost_budget"] = cb.summary()
	}
	for _, r := range all {
		delete(r, "after")
		delete(r, dirtyKey)
		delete(r, perDivKey)
		delete(r, ehpKey)
		delete(r, worstBaseKey)
	}
	if len(empty) > 0 {
		res["empty_parts"] = empty
	}
	if len(rz.forced) > 0 {
		res["forced_conditions"] = rz.forced
	}
	res["results"] = toAny(nonNil(all))
	res["elapsed_ms"] = time.Since(start).Milliseconds()
	if sc.partial {
		res["partial"] = true
	}
	res["hint"] = "Apply a row with scan_build apply=<its id> (dry_run previews); re-scan after each change."
	if cb != nil {
		if over := cb.overBudget(); over != "" {
			res["hint"] = over + ". " + res["hint"].(string)
		}
	}
	fitBudget(res, "results", capScan, "narrow parts or slots, or lower per_slot")
	return res, nil
}

// resistGap is each elemental resistance a change leaves under its cap, or deeper under it than before: the
// headroom after, negative.
func resistGap(base, after map[string]float64) map[string]float64 {
	out := map[string]float64{}
	for _, el := range []string{"Fire", "Cold", "Lightning"} {
		if a := after[el]; a < 0 && a < base[el] {
			out[el] = a
		}
	}
	return out
}

// row scores a change from its after calculation: gain, gain_pct and resist_gap.
func (sc *scanCtx) row(id, slot, name string, after *calculator.Environment, ch change) map[string]any {
	gain, off, def := rankGain(sc.ps, sc.key, sc.misc.Base(), after)
	row := map[string]any{"id": id, "slot": slot, "name": name, "gain": round(gain), changeKey: ch}
	if sc.cb != nil && sc.cb.rankBy == rankBySavings {
		row[ehpKey] = after.Player.Output["TotalEHP"] - sc.misc.Base().Player.Output["TotalEHP"]
	}
	putOffDef(row, sc.ps, off, def)
	if sc.base != 0 {
		row["gain_pct"] = round(gain / math.Abs(sc.base) * 100)
	}
	if g := resistGap(sc.room, headroom(after)); len(g) > 0 {
		row["resist_gap"] = g
	}
	return row
}

// keep drops rows that gain nothing (under rank_by savings, rows that save nothing) and, with keep_resists, rows that
// open a gap.
func (sc *scanCtx) keep(row map[string]any) bool {
	sc.seen++
	savings := sc.cb != nil && sc.cb.rankBy == rankBySavings
	if row["gain"].(float64) <= 0 && !savings {
		return false
	}
	sc.gaining++
	if sc.in.KeepResists && row["resist_gap"] != nil {
		sc.gapped++
		return false
	}
	if sc.cb != nil {
		if !sc.cb.admitChange(row, sc.b, row[changeKey].(change)) {
			sc.priced++
			return false
		}
		switch sc.cb.rankBy {
		case rankByGainPerDiv:
			row[perDivKey] = sc.cb.perDiv(row["gain"].(float64), row)
		case rankBySavings:
			ehp, _ := row[ehpKey].(float64)
			row[perDivKey] = savingsPerPct(row["cost"].(float64), lossPct(row["gain"].(float64), sc.base, ehp, sc.misc.Base().Player.Output["TotalEHP"], sc.key == "TotalEHP"))
		}
	}
	return true
}

// perDivKey holds a row's gain per divine, or under rank_by savings its savings per percent lost, while per-slot cuts
// rank by it.
const perDivKey = "\x00perdiv"

// ehpKey holds a row's TotalEHP change under rank_by savings until its change columns carry it.
const ehpKey = "\x00ehp"

// slotScore is what per-slot cuts rank a row by: its gain per divine under rank_by gain_per_div, its savings per
// percent lost under rank_by savings, else its gain.
func slotScore(r map[string]any) float64 {
	if v, ok := r[perDivKey].(float64); ok {
		if math.IsNaN(v) {
			return math.Inf(-1)
		}
		return v
	}
	return r["gain"].(float64)
}

var noCandidate = map[string]string{
	"uniques":     "no unique fits the scanned slots",
	"flasks":      "no flask candidate",
	"affixes":     "no rare or magic item in the scanned slots has an open prefix or suffix its mods can fill",
	"anoints":     "no anointable item, or every anoint is allocated already",
	"clusters":    "no socketed cluster jewel with a notable to swap",
	"reservation": "no mana source found",
	"sockets":     "every gem in the scanned slots sits in a socket of its colour, or in a white or abyssal one",
	"gems":        "no enabled gem can take another level or quality",
	"swaps":       "no enabled support, curse, herald or aura has another gem to swap in",
	"links":       "no four- or five-link item whose group fills its links can take another link, and no six-link to unlink",
}

// emptyReason says why a part returned no rows.
func (sc *scanCtx) emptyReason(part string) string {
	out := noCandidate[part]
	switch {
	case sc.seen == 0 && sc.partial:
		out = "ran out of time before scoring a candidate"
	case sc.seen == 0:
	case sc.gaining == 0:
		out = fmt.Sprintf("none of %d candidates raises %s", sc.seen, sc.key)
	case sc.gapped == sc.gaining:
		out = fmt.Sprintf("keep_resists dropped all %d candidates that raise %s", sc.gaining, sc.key)
	case sc.cb != nil && sc.cb.rankBy == rankBySavings && sc.gapped+sc.priced == sc.gaining:
		out = fmt.Sprintf("none of %d candidates saves chaos at known prices", sc.seen)
	case sc.gapped+sc.priced == sc.gaining:
		out = fmt.Sprintf("cost_budget dropped all %d candidates that raise %s (over budget or of unknown price)", sc.priced, sc.key)
	}
	if sc.seen > 0 && sc.partial {
		out += fmt.Sprintf(" (ran out of time after %d)", sc.seen)
	}
	return out
}

// rankWorst orders rows by their worst-case gain (their gain where they have none), uniques whose cost their flags
// name last, and drops those it leaves at or below 0, but for gem downgrades and unusable rows that gain in PoB (they
// rank last, saying why), naming the parts that loses every row of.
func rankWorst(rows []map[string]any, empty map[string]string) []map[string]any {
	score := func(r map[string]any) float64 {
		if w, ok := r["worst_gain"].(float64); ok {
			return w
		}
		return r["gain"].(float64)
	}
	parts := map[string]int{}
	var kept []map[string]any
	for _, r := range rows {
		p := r["part"].(string)
		if score(r) > 0 || r["downgrade"] == true || (r[dirtyKey] != nil && r["unusable"] != nil && r["gain"].(float64) > 0) {
			kept = append(kept, r)
			parts[p] = -1
		} else if parts[p] >= 0 {
			parts[p]++
		}
	}
	for p, n := range parts {
		if n > 0 {
			empty[p] = fmt.Sprintf("all %d rows gain nothing in the worst case", n)
		}
	}
	sort.SliceStable(kept, func(i, j int) bool {
		if di, dj := kept[i][dirtyKey] != nil, kept[j][dirtyKey] != nil; di != dj {
			return dj
		}
		return score(kept[i]) > score(kept[j])
	})
	return kept
}

// bestPerSlot keeps the n best rows of each slot, dirty rows last.
func bestPerSlot(rows []map[string]any, n int) []map[string]any {
	sort.SliceStable(rows, func(i, j int) bool {
		if di, dj := rows[i][dirtyKey] != nil, rows[j][dirtyKey] != nil; di != dj {
			return dj
		}
		return slotScore(rows[i]) > slotScore(rows[j])
	})
	count := map[string]int{}
	var out []map[string]any
	for _, r := range rows {
		delete(r, dirtyKey)
		sl := r["slot"].(string)
		if count[sl] >= n {
			continue
		}
		count[sl]++
		out = append(out, r)
	}
	return out
}

// uniquesPerDivRows is how many of a slot's uniques, best gain first, rank_by gain_per_div and savings price and keep
// per_slot of: a count, not a time, so identical scans return identical rows.
var uniquesPerDivRows = 200

// uniques ranks every PoB unique for each gear slot through rank_candidates' uniques path, keeping its order: clean
// rows before those whose cost their flags name.
func (sc *scanCtx) uniques(slots []string) []map[string]any {
	var out []map[string]any
	for _, slot := range slots {
		if !slices.Contains(gearSlots, slot) {
			continue
		}
		ranked, _, cut, _ := rankUniques(sc.b, sc.key, rankIn{Slot: slot, MaxLevelReq: sc.in.MaxLevelReq}, time.Now(), math.MaxInt, nil)
		sc.partial = sc.partial || cut
		old := &calculator.Item{}
		for _, e := range equippedItems(sc.b) {
			if e.slot == slot {
				old = calculator.ParseItem(e.text)
			}
		}
		gems := 0
		for _, g := range *groupsOf(sc.b) {
			if g.Source == "" && g.Slot == slot {
				gems += len(g.Gems)
			}
		}
		var rows []map[string]any
		perDiv := sc.cb != nil && sc.cb.rankBy != rankByGain
		for n, r := range ranked {
			if perDiv && n == uniquesPerDivRows {
				break
			}
			name := r["name"].(string)
			text, ok := calculator.ItemDBText(calculator.ItemDBUnique, name)
			if !ok {
				continue
			}
			item := calculator.ParseItem(text)
			// as rankUniques: PoB's comparison keeps the slot's socket colours and leaves out granted skills
			var after *calculator.Environment
			if coloursDiffer(old, item, gems) || len(item.GrantedSkills) > 0 {
				after = evalChange(sc.b, uniqueChange(text, slot))
			} else {
				after = sc.misc.Calc(&calculator.CalcOverride{RepSlotName: slot, RepItem: item})
			}
			row := sc.row("unique|"+slot+"|"+name, slot, name, after, uniqueChange(text, slot))
			if !sc.keep(row) {
				continue
			}
			if r[dirtyKey] != nil {
				row[dirtyKey] = true
			}
			rows = append(rows, row)
			if len(rows) == sc.in.PerSlot && !perDiv {
				break
			}
		}
		if perDiv {
			rows = bestPerSlot(rows, sc.in.PerSlot)
		}
		out = append(out, rows...)
	}
	return out
}

func uniqueChange(text, slot string) change {
	return func(c *pob.PathOfBuilding) { calculator.EquipItem(c, text, slot) }
}

// flaskTarget is the flask slot a new flask goes in: the first empty one, else the flask whose removal loses least of
// the stat the scan ranks by.
func (sc *scanCtx) flaskTarget() (string, string) {
	have := map[string]string{}
	for _, e := range equippedItems(sc.b) {
		if strings.HasPrefix(e.slot, "Flask") {
			have[e.slot] = e.name
		}
	}
	for _, sl := range flaskSlots {
		if _, ok := have[sl]; !ok {
			return sl, ""
		}
	}
	best, bestGain := "", math.Inf(-1)
	for _, sl := range flaskSlots {
		gain, _, _ := rankGain(sc.ps, sc.key, sc.misc.Base(), sc.misc.Calc(&calculator.CalcOverride{RepSlotName: sl}))
		if gain > bestGain {
			best, bestGain = sl, gain
		}
	}
	return best, have[best]
}

// flaskCandidates are PoB's unique flasks and a plain magic flask of each utility base.
func flaskCandidates(b *pob.PathOfBuilding) []struct{ name, text string } {
	var out []struct{ name, text string }
	equipped := map[string]bool{}
	for _, e := range equippedItems(b) {
		equipped[uniqueTitle(e.name)] = true
	}
	for _, it := range calculator.ItemDB(calculator.ItemDBUnique) {
		if it.Base == nil || it.Base.Flask == nil || equipped[uniqueTitle(it.Name)] {
			continue
		}
		if text, ok := calculator.ItemDBText(calculator.ItemDBUnique, it.Name); ok {
			out = append(out, struct{ name, text string }{it.Name, text})
		}
	}
	for _, name := range baseNames() {
		base := calculator.ItemBaseByName(name)
		if base == nil || base.Flask == nil || base.SubType != "Utility" || equipped[norm(name)] {
			continue
		}
		if text, ok := calculator.CraftItemText(1, "", name); ok {
			out = append(out, struct{ name, text string }{name, text})
		}
	}
	return out
}

// flaskSavingsTargets are the flask slots whose flask has a price above 0, and the flask's name: under rank_by
// savings each is a slot a cheaper flask can take.
func (sc *scanCtx) flaskSavingsTargets() [][2]string {
	var out [][2]string
	for _, e := range equippedItems(sc.b) {
		if pr := sc.cb.itemPrice(e.text); strings.HasPrefix(e.slot, "Flask") && pr.known() && *pr.Chaos > 0 {
			out = append(out, [2]string{e.slot, e.name})
		}
	}
	return out
}

func (sc *scanCtx) flasks(deadline time.Time) []map[string]any {
	var targets [][2]string
	if sc.cb != nil && sc.cb.rankBy == rankBySavings {
		targets = sc.flaskSavingsTargets()
	} else if slot, replaces := sc.flaskTarget(); slot != "" {
		targets = [][2]string{{slot, replaces}}
	}
	rows := make([]map[string]any, 0, len(targets))
	for _, t := range targets {
		rows = append(rows, sc.flasksInto(t[0], t[1], deadline)...)
	}
	return bestPerSlot(bestSlotPerName(rows), sc.in.PerSlot+1)
}

// bestSlotPerName keeps, of the rows that put one flask in several slots, the one that scores best.
func bestSlotPerName(rows []map[string]any) []map[string]any {
	best := map[string]int{}
	for i, r := range rows {
		n := r["name"].(string)
		if b, ok := best[n]; !ok || slotScore(r) > slotScore(rows[b]) {
			best[n] = i
		}
	}
	out := make([]map[string]any, 0, len(best))
	for i, r := range rows {
		if best[r["name"].(string)] == i {
			out = append(out, r)
		}
	}
	return out
}

// flasksInto scores each flask candidate in slot, replacing the flask named replaces there (none when empty).
func (sc *scanCtx) flasksInto(slot, replaces string, deadline time.Time) []map[string]any {
	var rows []map[string]any
	for _, c := range flaskCandidates(sc.b) {
		if time.Now().After(deadline) {
			sc.partial = true
			break
		}
		item := calculator.ParseItem(c.text)
		if item.Base == nil {
			continue
		}
		override := &calculator.CalcOverride{RepSlotName: slot, RepItem: item}
		if replaces == "" || !flaskActive(sc.b, slot) {
			// the row's change activates the flask; an inactive one it replaces adds nothing
			override = &calculator.CalcOverride{ToggleFlask: item}
		}
		row := sc.row("flask|"+slot+"|"+c.name, slot, c.name, sc.misc.Calc(override), flaskChange(c.text, slot))
		if replaces != "" {
			row["replaces"] = replaces
		}
		if f := flaskFlags(item); len(f) > 0 {
			row["flags"] = f
		}
		if sc.keep(row) {
			rows = append(rows, row)
		}
	}
	return rows
}

// flaskFlags names what a flask's gain leaves out: an effect under half its base's duration, which PoB counts as on
// all the time, and the lines PoB does not model, drawbacks such as Petrified during Effect among them.
func flaskFlags(item *calculator.Item) []string {
	var out []string
	if d, base := item.FlaskData["duration"], item.Base.Flask.Duration; d > 0 && d < base/2 {
		out = append(out, fmt.Sprintf("uptime: the effect lasts %gs a use (the base's %gs); PoB counts the flask as always on", d, base))
	}
	for _, ml := range slices.Concat(item.EnchantModLines, item.ImplicitModLines, item.ExplicitModLines) {
		if ml.Extra != "" {
			out = append(out, "unmodelled: "+ml.Line)
		}
	}
	return out
}

// flaskActive reports whether the flask in slot is switched on.
func flaskActive(b *pob.PathOfBuilding, slot string) bool {
	for _, e := range equippedItems(b) {
		if e.slot == slot {
			return e.active
		}
	}
	return false
}

func flaskChange(text, slot string) change {
	return func(c *pob.PathOfBuilding) {
		calculator.EquipItem(c, text, slot)
		c.SetFlaskActive(slot, true)
	}
}

// anoints ranks PoB's anointable notables (NotableDBControl.lua:17-19) on each anointable equipped item, replacing its
// first anoint as PoB's anoint popup does (ItemsTab.lua:3011-3023); notables with stat lines PoB cannot parse are
// returned unscored.
func (sc *scanCtx) anoints(deadline time.Time) ([]map[string]any, []string) {
	var targets []equipped
	for _, e := range equippedItems(sc.b) {
		item := calculator.ParseItem(e.text)
		if item.Base != nil && (item.Base.Type == "Amulet" || item.CanBeAnointed) {
			targets = append(targets, e)
		}
	}
	var rows []map[string]any
	var unmodelled []string
	for _, t := range targets {
		current := calculator.ItemAnoints(calculator.ParseItem(t.text))
		for _, n := range calculator.AnointNotables() {
			if slices.Contains(sc.b.Build.PassiveNodes, n.ID) || slices.Contains(current, n.Name) {
				continue
			}
			if slices.Contains(n.Unsupported, true) {
				if !slices.Contains(unmodelled, n.Name) {
					unmodelled = append(unmodelled, n.Name)
				}
				continue
			}
			if time.Now().After(deadline) {
				sc.partial = true
				break
			}
			text, err := calculator.EditItem(t.text, calculator.ItemEdit{Kind: "anoint", Index: 1, Text: n.Name})
			if err != nil {
				continue
			}
			after := sc.misc.Calc(&calculator.CalcOverride{RepSlotName: t.slot, RepItem: calculator.ParseItem(text)})
			row := sc.row("anoint|"+t.slot+"|"+n.Name, t.slot, n.Name, after, uniqueChange(text, t.slot))
			if len(current) > 0 {
				row["replaces"] = current[0]
			}
			if sc.keep(row) {
				rows = append(rows, row)
			}
		}
	}
	return bestPerSlot(bestSlotPerName(rows), sc.in.PerSlot+1), unmodelled
}

const schemaScanBuild = `{"type":"object","required":["handle"],"additionalProperties":false,"properties":{
  "handle":{"type":"string"},
  "stat":{"type":"string","default":"MainDPS","description":"stat key every part ranks by (reservation sources rank by ManaUnreserved; OffenceDefence ranks parts:[\"flasks\"] alone)"},
  "parts":{"type":"array","items":{"type":"string","enum":["uniques","flasks","affixes","anoints","clusters","reservation","resists","sockets","links","gems","swaps"]},"description":"default gems, uniques, flasks, affixes, anoints; resists plans the fewest resistance suffixes in open slots that cap fire, cold and lightning; sockets recolours each gear slot's sockets to its gems' colours; links adds a link to a four- or five-link item whose group fills its links, with each support that can support the group (linkup rows, priced at the bench's link cost and the support; under rank_by savings it instead takes a six-link to five links); gems sets each enabled gem to its maximum level or 20 quality, a corrupted 1 level or 23 quality more, or level and quality together (per_slot rows per socket group, per gem under rank_by gain_per_div); swaps replaces one gem of a socket group with another it can use, each support with a support PoB lets support the group (tried against the 8 that gain most in place of its weakest support) and each curse, herald or aura with an unused one of its kind (gemswap rows, per_slot per group)"},
  "downgrades":{"type":"boolean","default":false,"description":"gems: also list every downgrade of a corrupted level or quality back to the uncorrupted maximum, and a gem's quality back to 0 where it changes nothing (downgrade: true, kept whatever it loses; its cost under cost_budget is negative, what it saves); on under rank_by savings"},
  "slots":{"type":"array","items":{"type":"string"},"description":"limit uniques, affixes and sockets to these gear slots; affixes also scans magic flasks (Flask 1-5) unless slots narrows it"},
  "per_slot":{"type":"integer","default":2,"maximum":5},
  "keep_resists":{"type":"boolean","default":false,"description":"drop rows that leave an elemental resistance under its cap"},
  "max_level_req":{"type":"integer"},
  "mana_needed":{"type":"number","description":"reservation: extra unreserved mana wanted; ranks where it comes from"},
  "max_points":{"type":"integer","default":3,"description":"reservation: longest tree path considered"},
  "apply":{"type":"string","description":"a row id from an earlier scan: apply that change (other options ignored)"},
  "dry_run":{"type":"boolean","default":false},
  "watch":{"type":"array","items":{"type":"string"},"description":"apply: extra stats in the delta"},
  "roll":{"type":"string","enum":["min","avg","max"],"default":"avg","description":"affixes: where in its range an added mod rolls; avg is PoB's default for a mod without a stored roll"},
  "cap_chaos":{"type":"boolean","default":false,"description":"resists: cap chaos resistance too"},
  "delve":{"type":"boolean","default":false,"description":"affixes: also offer Delve drop-restricted and fossil-only mods (Subterranean, of the Underground), which only drop on Delve items or roll from fossils and cannot be added to an existing item"},
  "tier":{"type":"integer","minimum":1,"description":"affixes: add this tier (1 = the base's best) instead of the best the item level allows; groups without it, or above the item level, are skipped"},` + schemaBudgetProps + `}}`
