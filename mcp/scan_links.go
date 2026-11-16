package mcp

import (
	"fmt"
	"slices"
	"time"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
)

// linkable is an equipped item whose sockets are one linked group of four or five gem sockets below its base's socket
// limit, not corrupted or mirrored, and the enabled groups in its slot whose enabled gems fill those links.
type linkable struct {
	e      equipped
	links  int
	groups []int
}

func linkables(b *pob.PathOfBuilding, slots []string) []linkable {
	var out []linkable
	for _, e := range equippedItems(b) {
		if !slices.Contains(slots, e.slot) {
			continue
		}
		item := calculator.ParseItem(e.text)
		n := len(item.Sockets)
		if item.Base == nil || item.Corrupted || item.Mirrored || n < 4 || n > 5 || n >= item.Base.SocketLimit || maxLinks(item) != n ||
			slices.ContainsFunc(item.Sockets, func(s calculator.ItemSocket) bool { return !slices.Contains([]string{"R", "G", "B", "W"}, s.Color) }) {
			continue
		}
		var groups []int
		for gi, g := range *groupsOf(b) {
			if g.Slot != e.slot || g.Source != "" || !g.Enabled {
				continue
			}
			on := 0
			for _, gem := range g.Gems {
				if gem.Enabled {
					on++
				}
			}
			if on == n {
				groups = append(groups, gi+1)
			}
		}
		if len(groups) > 0 {
			out = append(out, linkable{e, n, groups})
		}
	}
	return out
}

// linkUpText is the item text with one more socket, linked to the others, in the colour of the gem it takes.
func linkUpText(text string, gem pob.Gem) string {
	sockets := slices.Clone(calculator.ParseItem(text).Sockets)
	colour := "W"
	if sk, ok := raw.PoBSkills[gemSkillID(gem)]; ok && sk.Color >= 1 && sk.Color <= 3 {
		colour = []string{"R", "G", "B"}[sk.Color-1]
	}
	sockets = append(sockets, calculator.ItemSocket{Color: colour, Group: sockets[len(sockets)-1].Group})
	return setSockets(text, socketLine(sockets))
}

// linkUpChange sets item id one link higher and adds gem to group (1-based).
func linkUpChange(id int, text string, group int, gem pob.Gem) change {
	next := linkUpText(text, gem)
	return func(c *pob.PathOfBuilding) {
		replaceItemText(id, next)(c)
		g := &(*groupsOf(c))[group-1]
		g.Gems = append(slices.Clone(g.Gems), gem)
	}
}

// linkUps ranks adding a link to a four- or five-link item and a support to the group that fills it: each support
// that can support the group's skills, by a full calculation of the build with the linked item and the support (the
// support PoB adds, at the build's default level and quality). The item is priced one link up (a rare's bench cost) and
// the support is bought.
func (sc *scanCtx) linkUps(slots []string, deadline time.Time) []map[string]any {
	var rows []map[string]any
	for _, l := range linkables(sc.b, slots) {
		for _, gi := range l.groups {
			g := (*groupsOf(sc.b))[gi-1]
			var candidates []gemEntry
			for _, ge := range gems() {
				if ge.Support && !slices.ContainsFunc(g.Gems, func(x pob.Gem) bool { return gemMatches(x, ge.Name) }) {
					candidates = append(candidates, ge)
				}
			}
			compat := supportCompat(sc.env, gi, candidates)
			for i, ge := range candidates {
				if !compat[i] {
					continue
				}
				if time.Now().After(deadline) {
					sc.partial = true
					return rows
				}
				gem := addedGem(sc.b, ge, nil, nil)
				ch := linkUpChange(l.e.id, l.e.text, gi, gem)
				name := fmt.Sprintf("%s %dL with %s", l.e.name, l.links+1, ge.Name)
				row := sc.row(fmt.Sprintf("linkup|%s|%d|%s", l.e.slot, gi, ge.Name), l.e.slot, name, evalChange(sc.b, ch), ch)
				row["adds"] = ge.Name
				if sc.keep(row) {
					rows = append(rows, row)
				}
			}
		}
	}
	return rows
}

// linkUpScanChange rebuilds a linkup row's change: slot's item one link up, the support named gem added to group
// (1-based).
func linkUpScanChange(b *pob.PathOfBuilding, slot string, group int, gem string) (change, string, *ToolError) {
	for _, l := range linkables(b, []string{slot}) {
		if !slices.Contains(l.groups, group) {
			continue
		}
		at := slices.IndexFunc(gems(), func(ge gemEntry) bool { return ge.Support && ge.Name == gem })
		if at < 0 {
			return nil, "", notFound("support gem", gem, nil, "Re-run scan_build and use an id from its results.")
		}
		return linkUpChange(l.e.id, l.e.text, group, addedGem(b, gems()[at], nil, nil)), fmt.Sprintf("link %s to %d links and add %s to group %d", slot, l.links+1, gem, group), nil
	}
	return nil, "", errf(CodeInvalidValue, "%s holds no four- or five-link that can take another link with group %d filling it.", slot, group)
}
