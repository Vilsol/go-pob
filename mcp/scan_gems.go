package mcp

import (
	"fmt"
	"strings"
	"time"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

// gemTry is one level or quality a gem can be set to instead of its own.
type gemTry struct {
	id, name string
	pair     gemPair
}

// gemTries are the levels and qualities a gem can legally hold at character level charLevel (no level whose requirement
// is above it), a corrupted copy included: its natural maximum level (or the highest the character can use),
// 20 quality, one level more or quality up to 23 when that is its one corruption (legalGem), and both at once (the
// natural or one more level at 20 quality, the natural level at 23); and, for a gem whose corruption is in its level
// or quality, that level or quality back to the uncorrupted maximum.
func gemTries(gi, j int, gem pob.Gem, charLevel int) []gemTry {
	natural := 20
	if ge, e := resolveGem(gem.NameSpec); e == nil && ge.MaxLevel > 0 {
		natural = ge.MaxLevel
	}
	skill := gemSkillID(gem)
	top := usableGemLevel(skill, natural, charLevel)
	awakened := strings.HasPrefix(gem.NameSpec, "Awakened ")
	cur := gemPair{gem.Level, gem.Quality}
	var out []gemTry
	add := func(p gemPair) {
		if p == cur || !legalGem(p, natural, awakened, true) {
			return
		}
		for _, t := range out {
			if t.pair == p {
				return
			}
		}
		name := fmt.Sprintf("%s %d/%d -> %d/%d", gem.NameSpec, cur.level, cur.quality, p.level, p.quality)
		switch {
		case p.level != cur.level && p.quality != cur.quality:
			out = append(out, gemTry{fmt.Sprintf("gemlevel|%d|%d|%d|%d", gi, j+1, p.level, p.quality), name, p})
		case p.level != cur.level:
			out = append(out, gemTry{fmt.Sprintf("gemlevel|%d|%d|%d", gi, j+1, p.level), name, p})
		default:
			out = append(out, gemTry{fmt.Sprintf("gemquality|%d|%d|%d", gi, j+1, p.quality), name, p})
		}
	}
	add(gemPair{top, cur.quality})
	add(gemPair{cur.level, gemQualityMax})
	add(gemPair{cur.level, gemQualityCorruptedMax})
	add(gemPair{top, gemQualityMax})
	add(gemPair{top, gemQualityCorruptedMax})
	if top == natural && gemLevelReq(skill, natural+1) <= charLevel {
		add(gemPair{natural + 1, cur.quality})
		add(gemPair{natural + 1, gemQualityMax})
	}
	return out
}

// gems ranks each enabled gem's level and quality changes that gain, the per_slot best of each socket group (under
// rank_by gain_per_div, of each gem, so a gem's dearer upgrade is not cut for another gem's cheap ones); and, with
// downgrades or under rank_by savings, every downgrade that takes a corrupted level or quality back to the uncorrupted
// maximum, whatever it loses, and every gem's quality back to 0 where that changes neither the scan's stat nor
// idleOutputs, marked downgrade (its cost, under cost_budget, is what it saves, negative).
func (sc *scanCtx) gems(deadline time.Time) []map[string]any {
	var ups, downs []map[string]any
	downgrades := sc.in.Downgrades || (sc.cb != nil && sc.cb.rankBy == rankBySavings)
	cut := func() []map[string]any {
		return append(sc.bestPerGem(ups), downs...)
	}
	for gi, g := range *groupsOf(sc.b) {
		if !g.Enabled || g.Source != "" {
			continue
		}
		for j, gem := range g.Gems {
			if !gem.Enabled {
				continue
			}
			for _, t := range gemTries(gi+1, j, gem, sc.levelCap()) {
				up := t.pair.level > gem.Level || t.pair.quality > gem.Quality
				if !up && !downgrades {
					continue
				}
				if time.Now().After(deadline) {
					sc.partial = true
					return cut()
				}
				ch := gemLevelChange(gi+1, j, t.pair.level, t.pair.quality)
				row := sc.row(t.id, fmt.Sprintf("group %d", gi+1), t.name, evalChange(sc.b, ch), ch)
				if up {
					row[gemKey] = fmt.Sprintf("%d|%d", gi+1, j+1)
					if sc.keep(row) {
						ups = append(ups, row)
					}
					continue
				}
				sc.seen++
				row["downgrade"] = true
				if sc.cb != nil && !sc.cb.admitChange(row, sc.b, ch) {
					sc.priced++
					continue
				}
				downs = append(downs, row)
			}
			if !downgrades || gem.Quality <= 0 || gem.Quality > gemQualityMax {
				continue
			}
			if time.Now().After(deadline) {
				sc.partial = true
				return cut()
			}
			ch := gemLevelChange(gi+1, j, gem.Level, 0)
			after := evalChange(sc.b, ch)
			sc.seen++
			if !sameOutputs(sc.misc.Base(), after, sc.key) {
				continue
			}
			row := sc.row(fmt.Sprintf("gemquality|%d|%d|0", gi+1, j+1), fmt.Sprintf("group %d", gi+1), fmt.Sprintf("%s %d/%d -> %d/0 (its quality changes nothing)", gem.NameSpec, gem.Level, gem.Quality, gem.Level), after, ch)
			row["downgrade"] = true
			if sc.cb != nil && !sc.cb.admitChange(row, sc.b, ch) {
				sc.priced++
				continue
			}
			downs = append(downs, row)
		}
	}
	return cut()
}

// idleOutputs are the outputs a gem's quality must leave as they are, besides the scan's stat, for dropping it to
// change nothing.
var idleOutputs = []string{"FullDPS", "CombinedDPS", "TotalEHP", "Life", "EnergyShield", "Mana", "LifeUnreserved", "ManaUnreserved", "Armour", "Evasion", "EffectiveMovementSpeedMod"}

// sameOutputs reports whether after leaves key and idleOutputs where base has them.
func sameOutputs(base, after *calculator.Environment, key string) bool {
	if statOf(base, key) != statOf(after, key) {
		return false
	}
	for _, k := range idleOutputs {
		if base.Player.Output[k] != after.Player.Output[k] {
			return false
		}
	}
	return true
}

// levelCap is the character level gem levels a scan proposes must be usable at: max_level_req when given, else the
// build's.
func (sc *scanCtx) levelCap() int {
	if sc.in.MaxLevelReq != nil {
		return *sc.in.MaxLevelReq
	}
	return characterLevel(sc.b)
}

// gemKey holds an upgrade row's group and gem while gems cuts rows per gem.
const gemKey = "\x00gem"

// bestPerGem keeps the per_slot best upgrade rows of each socket group, or under rank_by gain_per_div of each gem.
func (sc *scanCtx) bestPerGem(rows []map[string]any) []map[string]any {
	if sc.cb == nil || sc.cb.rankBy != rankByGainPerDiv {
		for _, r := range rows {
			delete(r, gemKey)
		}
		return bestPerSlot(rows, sc.in.PerSlot)
	}
	for _, r := range rows {
		r["slot"], r[gemKey] = r[gemKey], r["slot"]
	}
	out := bestPerSlot(rows, sc.in.PerSlot)
	for _, r := range out {
		r["slot"] = r[gemKey]
		delete(r, gemKey)
	}
	return out
}
