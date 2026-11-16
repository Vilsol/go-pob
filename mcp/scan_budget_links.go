package mcp

import (
	"fmt"
	"slices"
	"time"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

// fiveLink is a six-link item's sockets with the last one unlinked.
func fiveLink(sockets []calculator.ItemSocket) string {
	out := slices.Clone(sockets)
	last := len(out) - 1
	out[last].Group = out[last-1].Group + 1
	return socketLine(out)
}

// linksChange sets item id's text and drops gem (0-based) of group (1-based).
func linksChange(id int, text string, group, gem int) change {
	return func(c *pob.PathOfBuilding) {
		replaceItemText(id, text)(c)
		gemRemoveChange(group, gem)(c)
	}
}

// sixLinked is each equipped item in slots with a six-link, and the enabled groups of six or more gems it holds.
func sixLinked(b *pob.PathOfBuilding, slots []string) []struct {
	e      equipped
	groups []int
} {
	var out []struct {
		e      equipped
		groups []int
	}
	for _, e := range equippedItems(b) {
		if !slices.Contains(slots, e.slot) || maxLinks(calculator.ParseItem(e.text)) < 6 {
			continue
		}
		var groups []int
		for gi, g := range *groupsOf(b) {
			if g.Slot == e.slot && g.Source == "" && g.Enabled && len(g.Gems) >= 6 {
				groups = append(groups, gi+1)
			}
		}
		if len(groups) > 0 {
			out = append(out, struct {
				e      equipped
				groups []int
			}{e, groups})
		}
	}
	return out
}

// links ranks taking a six-link item down to five links, one support of its six-gem group dropped, by a full
// calculation; the item is priced at five links (a rare's bench cost, a unique's five-link listing) and the support
// is no longer bought. Unless ranking by savings it also ranks linking a four- or five-link item one link up (linkUps).
func (sc *scanCtx) links(slots []string, deadline time.Time) []map[string]any {
	var rows []map[string]any
	for _, six := range sixLinked(sc.b, slots) {
		text := setSockets(six.e.text, fiveLink(calculator.ParseItem(six.e.text).Sockets))
		for _, gi := range six.groups {
			for i, gem := range (*groupsOf(sc.b))[gi-1].Gems {
				if !gem.Enabled || !gemIsSupport(gem) {
					continue
				}
				if time.Now().After(deadline) {
					sc.partial = true
					return bestPerSlot(rows, sc.in.PerSlot)
				}
				ch := linksChange(six.e.id, text, gi, i)
				row := sc.row(fmt.Sprintf("links|%s|%d|%d", six.e.slot, gi, i+1), six.e.slot, six.e.name+" 5L without "+gem.NameSpec, evalChange(sc.b, ch), ch)
				row["drops"] = gem.NameSpec
				if sc.keep(row) {
					rows = append(rows, row)
				}
			}
		}
	}
	if sc.cb == nil || sc.cb.rankBy != rankBySavings {
		rows = append(rows, sc.linkUps(slots, deadline)...)
	}
	return bestPerSlot(rows, sc.in.PerSlot)
}

// linksScanChange rebuilds a links row's change: slot's item at five links without the group's gem (both 1-based).
func linksScanChange(b *pob.PathOfBuilding, slot string, group, gem int) (change, string, *ToolError) {
	if six := sixLinked(b, []string{slot}); len(six) == 1 && slices.Contains(six[0].groups, group) && gem >= 1 && gem <= len((*groupsOf(b))[group-1].Gems) {
		name := (*groupsOf(b))[group-1].Gems[gem-1].NameSpec
		text := setSockets(six[0].e.text, fiveLink(calculator.ParseItem(six[0].e.text).Sockets))
		return linksChange(six[0].e.id, text, group, gem-1), fmt.Sprintf("unlink the sixth socket of %s and remove %s", slot, name), nil
	}
	return nil, "", errf(CodeInvalidValue, "%s holds no six-link with a six-gem group %d gem %d.", slot, group, gem)
}
