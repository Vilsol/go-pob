package mcp

import (
	"slices"
	"strings"
	"time"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
)

// socketRecolour is the item in e's slot with its red, green and blue sockets recoloured to the colour of the gem PoB
// places in each: the slot's groups fill its sockets in order, each continuing where the previous one ended, and a gem
// gains the matching-socket quality when its socket has its colour (SkillsTab.lua:1244-1270). White and abyssal
// sockets, the socket count and the links stay. It returns the sockets before and after and how many changed; ok is
// false when none does.
func socketRecolour(b *pob.PathOfBuilding, e equipped) (string, string, int, bool) {
	item := calculator.ParseItem(e.text)
	if item.Base == nil || len(item.Sockets) == 0 || item.SocketColourAlwaysMatches {
		return "", "", 0, false
	}
	sockets := slices.Clone(item.Sockets)
	changed, offset := 0, 0
	for _, g := range activeSkills(b) {
		if g.Slot != e.slot {
			continue
		}
		for i, gem := range g.Gems {
			idx := offset + i
			if idx >= len(sockets) {
				break
			}
			sk, ok := raw.PoBSkills[gemSkillID(gem)]
			if !ok || sk.Color < 1 || sk.Color > 3 {
				continue
			}
			colour := []string{"R", "G", "B"}[sk.Color-1]
			if c := sockets[idx].Color; c != colour && (c == "R" || c == "G" || c == "B") {
				sockets[idx].Color = colour
				changed++
			}
		}
		offset += len(g.Gems)
	}
	if changed == 0 {
		return "", "", 0, false
	}
	return socketLine(item.Sockets), socketLine(sockets), changed, true
}

// socketLine is the item text's "Sockets:" value: links as '-', groups apart as ' '.
func socketLine(sockets []calculator.ItemSocket) string {
	var sb strings.Builder
	for i, so := range sockets {
		if i > 0 {
			if so.Group == sockets[i-1].Group {
				sb.WriteByte('-')
			} else {
				sb.WriteByte(' ')
			}
		}
		sb.WriteString(so.Color)
	}
	return sb.String()
}

// recolourChange sets the sockets of item id.
func recolourChange(b *pob.PathOfBuilding, id int, sockets string) change {
	for _, it := range b.Items.Items {
		if it.ID == id {
			return replaceItemText(id, setSockets(it.Text, sockets))
		}
	}
	return func(*pob.PathOfBuilding) {}
}

// sockets ranks recolouring each scanned slot's sockets to its gems' colours, by a full calculation of the build with
// the recoloured item.
func (sc *scanCtx) sockets(slots []string, deadline time.Time) []map[string]any {
	var rows []map[string]any
	for _, e := range equippedItems(sc.b) {
		if !slices.Contains(slots, e.slot) {
			continue
		}
		was, next, n, ok := socketRecolour(sc.b, e)
		if !ok {
			continue
		}
		if time.Now().After(deadline) {
			sc.partial = true
			return rows
		}
		ch := recolourChange(sc.b, e.id, next)
		row := sc.row("sockets|"+e.slot, e.slot, e.name, evalChange(sc.b, ch), ch)
		row["sockets"] = next
		row["was"] = was
		row["recoloured"] = n
		if sc.keep(row) {
			rows = append(rows, row)
		}
	}
	return rows
}
