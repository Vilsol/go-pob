package mcp

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

const (
	rankByGain       = "gain"
	rankByGainPerDiv = "gain_per_div"
	rankBySavings    = "savings"
	// minCost is the cost a change above 0 and below one chaos counts as in gain_per_div
	minCost = 1.0
	// minLossPct is the loss savings_per_pct divides by when a change loses less
	minLossPct = 1.0
	// buildCostKey is optimize's synthetic stat for the build's price in chaos
	buildCostKey = "BuildCost"
)

var budgetRe = regexp.MustCompile(`^\s*(\+)?\s*(\d+(?:\.\d+)?)\s*(c|chaos|d|div|divine|divines)?\s*$`)

// budgetIn is the pricing options scan_build and optimize share.
type budgetIn struct {
	CostBudget        json.RawMessage `json:"cost_budget"`
	AllowUnknownPrice bool            `json:"allow_unknown_price"`
	RankBy            string          `json:"rank_by"`
	FitBudget         bool            `json:"fit_budget"`
}

// costBudget prices changes against a build: the parts a change adds cost their price, the parts it removes save
// theirs. Prices of item texts and gems are cached for the call.
type costBudget struct {
	s            *Server
	limit        *float64
	added        *float64
	div          float64
	allowUnknown bool
	rankBy       string
	fit          bool
	baseTotal    float64
	baseUnknown  int
	base         buildParts
	items        map[string]Price
	gems         map[string]Price
	// over, unpriced and noSaving count the changes dropped over budget, of unknown price, and (rank_by savings) saving
	// nothing
	over, unpriced, noSaving int
}

// buildParts is what a build pays for: each worn item's text by slot and each enabled gem by its priced identity.
type buildParts struct {
	items map[string]string
	gems  map[string]int
	gemOf map[string]pob.Gem
}

func partsOf(b *pob.PathOfBuilding) buildParts {
	p := buildParts{items: map[string]string{}, gems: map[string]int{}, gemOf: map[string]pob.Gem{}}
	texts := map[int]string{}
	for _, it := range b.Items.Items {
		texts[it.ID] = it.Text
	}
	for _, e := range equippedItems(b) {
		p.items[e.slot] = texts[e.id]
	}
	if spec := activeSpec(b); spec != nil {
		for _, s := range spec.Sockets {
			if text, ok := texts[s.ItemID]; ok && s.ItemID != 0 {
				p.items[fmt.Sprintf("Jewel %d", s.NodeID)] = text
			}
		}
	}
	for _, sk := range activeSkills(b) {
		if !sk.Enabled || sk.Source != "" {
			continue
		}
		for _, g := range sk.Gems {
			if !g.Enabled {
				continue
			}
			variant := ""
			if g.VariantID != nil {
				variant = *g.VariantID
			}
			k := fmt.Sprintf("%s|%s|%s|%d|%d", g.NameSpec, g.GemID, variant, g.Level, g.Quality)
			p.gems[k]++
			p.gemOf[k] = g
		}
	}
	return p
}

// newCostBudget reads the pricing options; nil when the call names none and need is false, so it prices nothing.
func (s *Server) newCostBudget(in budgetIn, b *pob.PathOfBuilding, need bool) (*costBudget, *ToolError) {
	rankBy := in.RankBy
	if rankBy == "" {
		rankBy = rankByGain
		if in.FitBudget {
			rankBy = rankBySavings
		}
	}
	if !slices.Contains([]string{rankByGain, rankByGainPerDiv, rankBySavings}, rankBy) {
		return nil, invalidArg(fmt.Sprintf("Unknown rank_by %q.", in.RankBy), []string{rankByGain, rankByGainPerDiv, rankBySavings})
	}
	set := len(in.CostBudget) > 0 && string(in.CostBudget) != "null"
	if in.FitBudget && (!set || rankBy != rankBySavings) {
		return nil, invalidArg("fit_budget needs an absolute cost_budget (e.g. \"5div\") and ranks by savings.", nil)
	}
	if !set && rankBy == rankByGain && !need {
		if in.AllowUnknownPrice {
			return nil, invalidArg("allow_unknown_price needs cost_budget or rank_by gain_per_div.", nil)
		}
		return nil, nil
	}
	var amount float64
	var perDiv, plus bool
	if set {
		var e *ToolError
		if amount, perDiv, plus, e = parseBudget(in.CostBudget); e != nil {
			return nil, e
		}
		if plus && in.FitBudget {
			return nil, invalidArg("fit_budget needs an absolute cost_budget: \"+<n>\" is spent on top of the build, which already fits it.", nil)
		}
	}
	if s.opts.Prices == nil {
		e := errf(CodePricesUnavailable, "cost_budget, rank_by gain_per_div or savings and BuildCost need prices; this server was started with --no-prices.")
		e.Hint = "Restart go-pob mcp without --no-prices, or with --prices <snapshot file>."
		return nil, e
	}
	div, ok := s.opts.Prices.Divine()
	if !ok {
		why := s.opts.Prices.Warning()
		if why == "" {
			why = "the price data has no Divine Orb rate"
		}
		return nil, errf(CodePricesUnavailable, "cost_budget, rank_by gain_per_div or savings and BuildCost need prices: %s.", why)
	}
	if perDiv {
		amount *= div
	}
	cb := &costBudget{s: s, div: div, allowUnknown: in.AllowUnknownPrice, rankBy: rankBy, fit: in.FitBudget, base: partsOf(b),
		items: map[string]Price{}, gems: map[string]Price{}}
	cost := PriceBuild(s.opts.Prices, b)
	cb.baseTotal, cb.baseUnknown = cost.TotalChaos, cost.Unknown
	if set {
		limit := amount
		if plus {
			limit += cb.baseTotal
			cb.added = &amount
		}
		cb.limit = &limit
	}
	return cb, nil
}

// parseBudget reads a chaos amount, or "<n>div" in divines; a leading "+" makes it an amount on top of the build.
func parseBudget(raw json.RawMessage) (float64, bool, bool, *ToolError) {
	var n float64
	if err := json.Unmarshal(raw, &n); err == nil {
		if n <= 0 {
			return 0, false, false, invalidArg("cost_budget must be positive.", nil)
		}
		return n, false, false, nil
	}
	var str string
	_ = json.Unmarshal(raw, &str)
	m := budgetRe.FindStringSubmatch(strings.ToLower(str))
	if m == nil {
		return 0, false, false, invalidArg(fmt.Sprintf("cost_budget %s is not a chaos amount or \"<n>div\".", raw), []string{"300", "\"300c\"", "\"5div\"", "\"+5div\""})
	}
	n, _ = strconv.ParseFloat(m[2], 64)
	if n <= 0 {
		return 0, false, false, invalidArg("cost_budget must be positive.", nil)
	}
	return n, strings.HasPrefix(m[3], "d"), m[1] == "+", nil
}

func (cb *costBudget) itemPrice(text string) Price {
	if p, ok := cb.items[text]; ok {
		return p
	}
	p := PriceItem(cb.s.opts.Prices, calculator.PeekItem(text))
	cb.items[text] = p
	return p
}

func (cb *costBudget) gemPrice(k string, g pob.Gem) Price {
	if p, ok := cb.gems[k]; ok {
		return p
	}
	p := PriceGem(cb.s.opts.Prices, g)
	cb.gems[k] = p
	return p
}

// delta is what after costs over the base build in chaos, negative for a saving, and the parts it adds whose price is
// unknown. A removed part of unknown price saves nothing. A new anoint on the same item needs oils, whose price the
// price data does not list.
func (cb *costBudget) delta(after *pob.PathOfBuilding) (float64, []string) {
	p := partsOf(after)
	cost := 0.0
	var unknown []string
	for slot, text := range p.items {
		was := cb.base.items[slot]
		if was == text {
			continue
		}
		item := calculator.PeekItem(text)
		if pr := cb.itemPrice(text); pr.known() {
			cost += *pr.Chaos
		} else {
			unknown = append(unknown, item.Name)
		}
		if was != "" && calculator.PeekItem(was).Name == item.Name {
			old := calculator.ItemAnoints(calculator.PeekItem(was))
			for _, a := range calculator.ItemAnoints(item) {
				if !slices.Contains(old, a) {
					unknown = append(unknown, "oils for "+a)
				}
			}
		}
	}
	for slot, text := range cb.base.items {
		if p.items[slot] == text {
			continue
		}
		if pr := cb.itemPrice(text); pr.known() {
			cost -= *pr.Chaos
		}
	}
	keys := map[string]bool{}
	for k := range p.gems {
		keys[k] = true
	}
	for k := range cb.base.gems {
		keys[k] = true
	}
	for k := range keys {
		n := p.gems[k] - cb.base.gems[k]
		if n == 0 {
			continue
		}
		g, ok := p.gemOf[k]
		if !ok {
			g = cb.base.gemOf[k]
		}
		pr := cb.gemPrice(k, g)
		switch {
		case pr.known():
			cost += float64(n) * *pr.Chaos
		case n > 0:
			unknown = append(unknown, fmt.Sprintf("%s %d/%d", g.NameSpec, g.Level, g.Quality))
		}
	}
	sort.Strings(unknown)
	return math.Round(cost*10) / 10, unknown
}

// admit prices a change: it puts cost, or cost_unknown and no cost, on row and reports whether the row stays: within
// the budget, and under rank_by savings saving chaos.
func (cb *costBudget) admit(row map[string]any, after *pob.PathOfBuilding) bool {
	cost, unknown := cb.delta(after)
	if len(unknown) > 0 {
		row["cost_unknown"] = unknown
		delete(row, "cost")
		if !cb.allowUnknown || cb.rankBy == rankBySavings {
			cb.unpriced++
			return false
		}
		return true
	}
	row["cost"] = cost
	if cb.rankBy == rankBySavings {
		if cost >= 0 {
			cb.noSaving++
			return false
		}
		return true
	}
	if cb.limit != nil && cb.baseTotal+cost > *cb.limit {
		cb.over++
		return false
	}
	return true
}

// admitChange prices a row's change applied to b.
func (cb *costBudget) admitChange(row map[string]any, b *pob.PathOfBuilding, ch change) bool {
	c := cloneBuild(b)
	ch(c)
	return cb.admit(row, c)
}

// perDivOf is gain per divine of cost: +Inf when the change gains and costs nothing or saves (it pays for itself); a
// cost above 0 and below minCost counts as minCost.
func perDivOf(gain, cost, div float64) float64 {
	if cost <= 0 && gain > 0 {
		return math.Inf(1)
	}
	return gain / (math.Max(cost, minCost) / div)
}

// perDiv is a row's gain per divine; NaN when its cost is unknown.
func (cb *costBudget) perDiv(gain float64, row map[string]any) float64 {
	if row["cost_unknown"] != nil {
		return math.NaN()
	}
	cost, _ := row["cost"].(float64)
	return perDivOf(gain, cost, cb.div)
}

// putPerDiv sets gain_per_div from score, the row's of (worst_gain or gain), or pays_for_itself. A row that costs
// nothing or saves pays for itself only when it is usable and loses nothing on the savings basis (lossBasis), in no
// watched stat (watchedLosses); one that does not gets loss_pct, savings_per_pct and watched_loss instead, as rank_by
// savings scores it, and no gain_per_div.
func (cb *costBudget) putPerDiv(row map[string]any, score float64, of string, base, baseEHP float64, ehpStat bool) {
	cost, _ := row["cost"].(float64)
	if row["cost_unknown"] == nil && cost <= 0 {
		watched := watchedLosses(row)
		if loss := basisLoss(row, base, baseEHP, ehpStat); score <= 0 || loss > 0 || len(watched) > 0 || row["unusable"] != nil {
			row["loss_pct"] = round(loss)
			row["loss_of"] = of
			if cost < 0 {
				row["savings_per_pct"] = round(savingsPerPct(cost, loss))
			}
			if len(watched) > 0 {
				row["watched_loss"] = watched
			}
			return
		}
	}
	switch v := cb.perDiv(score, row); {
	case math.IsNaN(v):
		return
	case math.IsInf(v, 1):
		row["pays_for_itself"] = true
	default:
		row["gain_per_div"] = round(v)
	}
	row["gain_per_div_of"] = of
}

// lossBasis is what a row's savings loss counts, all on one basis (gain, its stat base, TotalEHP change, TotalEHP
// base): with a worst_gain, the worst_gain, worst_d's TotalEHP change and the stat and TotalEHP of the calculation they
// start from; else the gain, d's TotalEHP change and base and baseEHP, the current build's.
func lossBasis(row map[string]any, base, baseEHP float64) (float64, float64, float64, float64) {
	if w, ok := row["worst_gain"].(float64); ok {
		wd, _ := row["worst_d"].(map[string]float64)
		if wb, ok := row[worstBaseKey].(map[string]float64); ok {
			return w, wb["stat"], wd["dEHP"], wb["TotalEHP"]
		}
		return w, base, wd["dEHP"], baseEHP
	}
	d, _ := row["d"].(map[string]float64)
	g, _ := row["gain"].(float64)
	return g, base, d["dEHP"], baseEHP
}

// basisLoss is lossPct on a row's lossBasis.
func basisLoss(row map[string]any, base, baseEHP float64, ehpStat bool) float64 {
	g, sb, dEHP, eb := lossBasis(row, base, baseEHP)
	return lossPct(g, sb, dEHP, eb, ehpStat)
}

// watchedColumns are the change columns of the stats a row is watched for losing.
var watchedColumns = []string{"dDPS", "dFullDPS", "dMainDPS", "dEHP", "dMaxHit", "dLife", "dTotemLife"}

// watchedLosses names what a row loses on its basis (worst_d where it has a worst_gain, else d): each watched column
// that falls, and the capped avoidances and immunities its flags say it loses.
func watchedLosses(row map[string]any) []string {
	cols, _ := row["d"].(map[string]float64)
	if _, ok := row["worst_gain"].(float64); ok {
		cols, _ = row["worst_d"].(map[string]float64)
	}
	var out []string
	for _, c := range watchedColumns {
		if v := cols[c]; v < 0 {
			out = append(out, fmt.Sprintf("%s %v", c, v))
		}
	}
	flags, _ := row["flags"].([]string)
	for _, f := range flags {
		if l, ok := strings.CutPrefix(f, "loses: "); ok {
			out = append(out, l)
		}
	}
	return out
}

// markBigLoss marks the rows whose big_loss names a watched stat the change takes down, so every ranking puts them
// after the clean rows and fit_budget passes them over.
func markBigLoss(rows []map[string]any) {
	for _, r := range rows {
		if r["big_loss"] != nil {
			r[dirtyKey] = true
		}
	}
}

// lossPct is the percentage a change loses: the ranked stat's fall (base, changed by gain) plus TotalEHP's (baseEHP,
// changed by dEHP), each counted only when it falls, so a gain in one does not hide a loss in the other. With ehpStat
// (the ranked stat is TotalEHP) only the stat counts. Rows pass both on their lossBasis.
func lossPct(gain, base, dEHP, baseEHP float64, ehpStat bool) float64 {
	loss := 0.0
	if base != 0 {
		loss += math.Max(-gain/math.Abs(base)*100, 0)
	}
	if baseEHP > 0 && !ehpStat {
		loss += math.Max(-dEHP/baseEHP*100, 0)
	}
	return loss
}

// savingsPerPct is the chaos a change saves per percent it loses (lossPct), a loss under minLossPct counting as
// minLossPct.
func savingsPerPct(cost, loss float64) float64 {
	return -cost / math.Max(loss, minLossPct)
}

const (
	gainPerDivNote = "gain_per_div is gain_per_div_of (worst_gain where a row has one, else gain) per divine of cost; a cost under 1 chaos counts as 1 chaos; a row that costs nothing or saves, is usable and loses nothing (loss_pct, as rank_by savings counts it, and no watched stat or capped avoidance down: watched_loss) is pays_for_itself and ranks first; one that loses shows loss_pct, savings_per_pct and watched_loss instead and ranks after the rows with a gain_per_div; rows with big_loss (" + bigLossNote + ") rank after the clean ones"
	savingsNote    = "savings_per_pct is the chaos a row saves (-cost) per percent it loses: loss_pct, the ranked stat's percentage fall plus TotalEHP's, each counted only when it falls, both on one basis (loss_of): worst_gain and worst_d against the worst-case build where the row has them, else gain and d against the current build; a loss under 1% counts as 1%; savings_per_pct equal to two significant figures tie and rank by loss_of's gain. Only rows that save chaos are kept; rows with big_loss (" + bigLossNote + ") rank after the clean ones and fit_budget passes them over"
	bigLossNote    = "a watched stat other than the ranked one (DPS, MainDPS or FullDPS, TotalEHP, lowest max hit, Life, TotemLife) down 20% or more, or a resistance down 20 points or more, on the row's basis: worst_d against the worst-case build where the row has a worst_gain, else d against the current build; unusable means only what the character cannot use (an unmet requirement, a cost it cannot pay, a disabled skill, broken sockets, off-hand or totem limit)"
)

// summary is the response's cost_budget block.
func (cb *costBudget) summary() map[string]any {
	out := map[string]any{"build_chaos": round(cb.baseTotal), "divine_chaos": cb.div, "rank_by": cb.rankBy}
	if cb.limit != nil {
		out["budget_chaos"] = round(*cb.limit)
		if cb.added != nil {
			out["added_chaos"] = round(*cb.added)
		}
		if cb.rankBy != rankBySavings {
			out["dropped_over_budget"] = cb.over
		}
		if over := cb.overBudget(); over != "" {
			out["over_budget"] = over
		}
	}
	if cb.baseUnknown > 0 {
		out["build_unknown"] = cb.baseUnknown
		out["note"] = fmt.Sprintf("build_chaos leaves out %d parts of unknown price", cb.baseUnknown)
	}
	if !cb.allowUnknown || cb.rankBy == rankBySavings {
		out["dropped_unknown_price"] = cb.unpriced
	}
	switch cb.rankBy {
	case rankByGainPerDiv:
		out["gain_per_div"] = gainPerDivNote
	case rankBySavings:
		out["dropped_no_saving"] = cb.noSaving
		out["savings"] = savingsNote
	}
	return out
}

// overBudget says that the build already costs more than an absolute cap; empty when it does not.
func (cb *costBudget) overBudget() string {
	if cb.limit == nil || cb.added != nil || cb.baseTotal <= *cb.limit {
		return ""
	}
	out := fmt.Sprintf("the build already costs %v chaos (%.1f div), %v over the %v chaos cap, so nothing that adds cost fits",
		round(cb.baseTotal), cb.baseTotal/cb.div, round(cb.baseTotal-*cb.limit), round(*cb.limit))
	if cb.rankBy != rankBySavings {
		out += "; rank_by \"savings\" ranks downgrades, fit_budget:true picks a set that brings the build under the cap, and \"+<n>div\" spends on top of the current build"
	}
	return out
}

// rowScore is what a row ranks by: its worst_gain where it has one, else its gain, and which one it is.
func rowScore(r map[string]any) (float64, string) {
	if w, ok := r["worst_gain"].(float64); ok {
		return w, "worst_gain"
	}
	return r["gain"].(float64), "gain"
}

// sortPerDiv orders rows by gain_per_div, unusable rows after clean ones, rows that pay for themselves first and those
// of unknown price last, keeping the order of ties.
func sortPerDiv(rows []map[string]any) {
	key := func(r map[string]any) float64 {
		if r["pays_for_itself"] != nil {
			return math.Inf(1)
		}
		if v, ok := r["gain_per_div"].(float64); ok {
			return v
		}
		return math.Inf(-1)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if di, dj := rows[i][dirtyKey] != nil, rows[j][dirtyKey] != nil; di != dj {
			return dj
		}
		a, b := key(rows[i]), key(rows[j])
		if a == b && math.IsInf(a, 1) {
			sa, _ := rowScore(rows[i])
			sb, _ := rowScore(rows[j])
			return sa > sb
		}
		return a > b
	})
}

// rankSavings scores rows by savings_per_pct, their loss on their lossBasis, and orders them by it (savingsTier, ties
// by worst_gain or gain), unusable rows and those with big_loss after clean ones.
func (cb *costBudget) rankSavings(rows []map[string]any, base, baseEHP float64, ehpStat bool) {
	for _, r := range rows {
		cost, _ := r["cost"].(float64)
		_, of := rowScore(r)
		loss := basisLoss(r, base, baseEHP, ehpStat)
		r["loss_pct"] = round(loss)
		r["loss_of"] = of
		r["savings_per_pct"] = round(savingsPerPct(cost, loss))
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if di, dj := rows[i][dirtyKey] != nil, rows[j][dirtyKey] != nil; di != dj {
			return dj
		}
		if a, b := savingsTier(rows[i]["savings_per_pct"].(float64)), savingsTier(rows[j]["savings_per_pct"].(float64)); a != b {
			return a > b
		}
		si, _ := rowScore(rows[i])
		sj, _ := rowScore(rows[j])
		if si != sj {
			return si > sj
		}
		return rows[i]["savings_per_pct"].(float64) > rows[j]["savings_per_pct"].(float64)
	})
}

// savingsTier is savings_per_pct to two significant figures: rows in one tier tie on it and rank by what they gain.
func savingsTier(v float64) float64 {
	if v == 0 || math.IsInf(v, 0) || math.IsNaN(v) {
		return v
	}
	p := math.Pow(10, math.Floor(math.Log10(math.Abs(v)))-1)
	return math.Round(v/p) * p
}

const schemaBudgetProps = `
  "cost_budget":{"type":["number","string"],"description":"chaos (300 or \"300c\") or divines (\"5div\"): drop results whose whole build would then cost more, at poe.ninja prices (price_build); \"+5div\" or \"+300c\" spends at most that on top of the current build. Results carry cost (chaos added, negative for a saving)"},
  "allow_unknown_price":{"type":"boolean","default":false,"description":"with cost_budget or rank_by gain_per_div: keep results that add an item or gem of unknown price (cost_unknown names it, and they carry no cost) instead of dropping them"},
  "rank_by":{"type":"string","enum":["gain","gain_per_div","savings"],"default":"gain","description":"gain_per_div ranks by gain per divine of cost (a cost under 1 chaos counts as 1 chaos; a row that gains and saves is pays_for_itself, first); savings keeps only rows that save chaos and ranks them by savings_per_pct, chaos saved per percent lost: the ranked stat's percentage fall plus TotalEHP's, each only when it falls, worst case where the row has one (a loss under 1% counts as 1%); rows with big_loss rank after the clean ones. Both need prices"},
  "fit_budget":{"type":"boolean","default":false,"description":"scan_build, with an absolute cost_budget: rank by savings and add fit_budget, the set of clean savings rows (at most one change per item slot and per gem) that brings the build under the cap at the least summed loss_pct, checked a pick at a time on the whole set: a pick that leaves the set with big_loss or a resistance gap the set's own open suffixes cannot refill (refill names them) is dropped; picks apply in the order listed; skipped counts the rows left out by reason; when no clean set fits, fits:false, no picks and closest_chaos"}`

// price puts a state's cost and BuildCost on ev.
func (cb *costBudget) price(ev *evaluation, c *pob.PathOfBuilding) {
	ev.cost, ev.unpriced = cb.delta(c)
	ev.vals[buildCostKey] = cb.baseTotal + ev.cost
}

// priceOut marks ev out when it breaks the budget, pushing the search away by lambda as a failed constraint does.
func (cb *costBudget) priceOut(ev *evaluation, lambda float64) {
	switch {
	case len(ev.unpriced) > 0 && !cb.allowUnknown:
		ev.score -= lambda
	case cb.limit != nil && cb.baseTotal+ev.cost > *cb.limit:
		ev.score -= lambda * (cb.baseTotal + ev.cost - *cb.limit) / math.Max(*cb.limit, 1)
	default:
		return
	}
	ev.priceOut, ev.feasible = true, false
}

// evalPerDiv is a state's gain per divine of cost; NaN when its cost is unknown.
func (cb *costBudget) evalPerDiv(ev *evaluation, gain float64) float64 {
	if len(ev.unpriced) > 0 {
		return math.NaN()
	}
	return perDivOf(gain, ev.cost, cb.div)
}
