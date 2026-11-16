package mcp

import (
	"fmt"
	"slices"
	"time"

	"github.com/Vilsol/go-pob/pob"
)

// swapKinds are the active gem kinds a swap keeps: a curse for another curse (a mark for a hex too), a herald for a
// herald, an aura for an aura.
var swapKinds = []string{"curse", "herald", "aura"}

// swapScreen is how many supports, best first when each replaces the group's weakest support, every support of the
// group is tried against: a count, not a time, so identical scans return identical rows.
var swapScreen = 8

func swapKind(g gemEntry) string {
	for _, k := range swapKinds {
		if slices.Contains(g.Tags, k) {
			return k
		}
	}
	return ""
}

// gemSwapChange puts the gem named ge in group's (1-based) slot at (0-based), as PoB's skills tab changes the gem in a
// slot.
func gemSwapChange(group, at int, ge gemEntry) change {
	return func(c *pob.PathOfBuilding) {
		g := &(*groupsOf(c))[group-1]
		g.Gems = slices.Clone(g.Gems)
		g.Gems[at] = replacedGem(c, g.Gems[at], ge, nil, nil)
	}
}

// swaps ranks, in each enabled socket group, replacing one gem with another the group can use: each enabled support with
// a support PoB lets support the group's skills, and each curse, herald or aura with an unused gem of its kind (a
// Vaal gem neither leaves nor joins). Supports are screened: every support is tried against the swapScreen supports
// that gain most in place of the group's weakest one. The new gem takes PoB's default level and the slot's quality;
// per_slot rows per group, by full calculation.
func (sc *scanCtx) swaps(deadline time.Time) []map[string]any {
	var rows []map[string]any
	inBuild := map[string]bool{}
	for _, g := range *groupsOf(sc.b) {
		if !g.Enabled {
			continue
		}
		for _, gem := range g.Gems {
			if gem.Enabled {
				inBuild[norm(gem.NameSpec)] = true
			}
		}
	}
	try := func(gi, at int, ge gemEntry) (map[string]any, bool) {
		if time.Now().After(deadline) {
			sc.partial = true
			return nil, false
		}
		old := (*groupsOf(sc.b))[gi-1].Gems[at]
		ch := gemSwapChange(gi, at, ge)
		row := sc.row(fmt.Sprintf("gemswap|%d|%d|%s", gi, at+1, ge.Name), fmt.Sprintf("group %d", gi), old.NameSpec+" -> "+ge.Name, evalChange(sc.b, ch), ch)
		row["replaces"], row["adds"] = old.NameSpec, ge.Name
		return row, true
	}
	var out []map[string]any
	for gi0, g := range *groupsOf(sc.b) {
		gi := gi0 + 1
		if !g.Enabled || g.Source != "" {
			continue
		}
		var supports []int
		for j, gem := range g.Gems {
			if !gem.Enabled {
				continue
			}
			if gemIsSupport(gem) {
				supports = append(supports, j)
				continue
			}
			ge, e := resolveGem(gem.NameSpec)
			kind := swapKind(ge)
			if e != nil || kind == "" || slices.Contains(ge.Tags, "vaal") {
				continue
			}
			for _, cand := range gems() {
				if cand.Support || inBuild[norm(cand.Name)] || swapKind(cand) != kind || slices.Contains(cand.Tags, "vaal") {
					continue
				}
				row, ok := try(gi, j, cand)
				if !ok {
					return append(out, bestPerSlot(rows, sc.in.PerSlot)...)
				}
				if sc.keep(row) {
					rows = append(rows, row)
				}
			}
		}
		if len(supports) > 0 {
			var all, candidates []gemEntry
			for _, ge := range gems() {
				if ge.Support && !slices.ContainsFunc(g.Gems, func(x pob.Gem) bool { return gemMatches(x, ge.Name) }) {
					all = append(all, ge)
				}
			}
			for i, ok := range supportCompat(sc.env, gi, all) {
				if ok {
					candidates = append(candidates, all[i])
				}
			}
			weakest, weakestGain := supports[0], 0.0
			for k, j := range supports {
				ch := gemRemoveChange(gi, j)
				gain, _, _ := rankGain(sc.ps, sc.key, sc.misc.Base(), evalChange(sc.b, ch))
				if k == 0 || gain > weakestGain {
					weakest, weakestGain = j, gain
				}
			}
			type screened struct {
				ge   gemEntry
				gain float64
			}
			var screen []screened
			for _, ge := range candidates {
				row, ok := try(gi, weakest, ge)
				if !ok {
					return append(out, bestPerSlot(rows, sc.in.PerSlot)...)
				}
				screen = append(screen, screened{ge, row["gain"].(float64)})
				if sc.keep(row) {
					rows = append(rows, row)
				}
			}
			slices.SortStableFunc(screen, func(a, b screened) int {
				switch {
				case a.gain > b.gain:
					return -1
				case a.gain < b.gain:
					return 1
				}
				return 0
			})
			for _, j := range supports {
				if j == weakest {
					continue
				}
				for _, s := range screen[:min(len(screen), swapScreen)] {
					row, ok := try(gi, j, s.ge)
					if !ok {
						return append(out, bestPerSlot(rows, sc.in.PerSlot)...)
					}
					if sc.keep(row) {
						rows = append(rows, row)
					}
				}
			}
		}
		out = append(out, bestPerSlot(rows, sc.in.PerSlot)...)
		rows = nil
	}
	return out
}

// gemSwapScanChange rebuilds a gemswap row's change: the gem in group's (1-based) slot at (1-based) replaced by name.
func gemSwapScanChange(b *pob.PathOfBuilding, group, at int, name string) (change, string, *ToolError) {
	groups := *groupsOf(b)
	if group < 1 || group > len(groups) || at < 1 || at > len(groups[group-1].Gems) {
		return nil, "", invalidArg("apply takes a row id from scan_build results, e.g. gemswap|1|2|Added Cold Damage.", nil)
	}
	ge, e := resolveGem(name)
	if e != nil {
		return nil, "", e
	}
	old := groups[group-1].Gems[at-1].NameSpec
	return gemSwapChange(group, at-1, ge), fmt.Sprintf("replace %s with %s in group %d", old, ge.Name, group), nil
}
