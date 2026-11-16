package mcp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

// scanChange rebuilds the change a scan row's id names from the build as it is now.
func scanChange(b *pob.PathOfBuilding, id string) (change, string, *ToolError) {
	parts := strings.Split(id, "|")
	bad := func() (change, string, *ToolError) {
		e := invalidArg("apply takes a row id from scan_build results, e.g. unique|Amulet|Astramentis.", nil)
		return nil, "", e
	}
	if len(parts) < 2 {
		return bad()
	}
	ints := func(from int) ([]int, bool) {
		var out []int
		for _, p := range parts[from:] {
			n, err := strconv.Atoi(p)
			if err != nil {
				return nil, false
			}
			out = append(out, n)
		}
		return out, true
	}
	slotText := func(slot string) (string, bool) {
		for _, e := range equippedItems(b) {
			if e.slot == slot {
				return e.text, true
			}
		}
		return "", false
	}
	switch parts[0] {
	case "unique", "flask":
		if len(parts) != 3 {
			return bad()
		}
		slot, name := parts[1], parts[2]
		text, ok := calculator.ItemDBText(calculator.ItemDBUnique, name)
		if !ok && parts[0] == "flask" {
			text, ok = calculator.CraftItemText(1, "", name)
		}
		if !ok {
			return nil, "", notFound("item", name, nil, "Re-run scan_build and use an id from its results.")
		}
		if parts[0] == "flask" {
			return flaskChange(text, slot), "equip " + name + " in " + slot + " (active)", nil
		}
		return uniqueChange(text, slot), "equip " + name + " in " + slot, nil
	case "anoint":
		if len(parts) != 3 {
			return bad()
		}
		text, ok := slotText(parts[1])
		if !ok {
			return nil, "", errf(CodeInvalidSlot, "Nothing equipped in %s.", parts[1])
		}
		next, err := calculator.EditItem(text, calculator.ItemEdit{Kind: "anoint", Index: 1, Text: parts[2]})
		if err != nil {
			return nil, "", errf(CodeInvalidValue, "Cannot anoint %s: %v", parts[1], err)
		}
		return uniqueChange(next, parts[1]), "anoint " + parts[1] + " with " + parts[2], nil
	case "affix":
		if len(parts) != 3 && len(parts) != 4 {
			return bad()
		}
		rng := defaultRoll.rng
		if len(parts) == 4 {
			r, err := strconv.ParseFloat(parts[3], 64)
			if err != nil || r < 0 || r > 1 {
				return bad()
			}
			rng = r
		}
		text, ok := slotText(parts[1])
		if !ok {
			return nil, "", errf(CodeInvalidSlot, "Nothing equipped in %s.", parts[1])
		}
		src, modID, _ := strings.Cut(parts[2], ":")
		next, ok := addModText(text, src, modID, rng)
		if !ok || !affixAllowed(calculator.ParseItem(text), src, modID) {
			return nil, "", errf(CodeAffixLimit, "%s cannot take %s.", parts[1], parts[2])
		}
		return uniqueChange(next, parts[1]), "add affix " + parts[2] + " to " + parts[1], nil
	case "resists":
		if len(parts) != 3 {
			return bad()
		}
		rng, err := strconv.ParseFloat(parts[1], 64)
		chaos, err2 := strconv.ParseBool(parts[2])
		if err != nil || err2 != nil {
			return bad()
		}
		items, picks, ok := resistPlan(b, headroom(calcEnv(b)), rng, chaos)
		if !ok || len(picks) == 0 {
			return nil, "", errf(CodeAffixLimit, "No set of resistance suffixes in the open slots caps the resistances.")
		}
		texts, ok := resistPlanTexts(items, picks, rng)
		if !ok {
			return nil, "", errf(CodeAffixLimit, "The resistance plan's mods do not fit the items.")
		}
		return resistPlanChange(texts), fmt.Sprintf("add %d resistance suffixes", len(picks)), nil
	case "sockets":
		if len(parts) != 2 {
			return bad()
		}
		for _, e := range equippedItems(b) {
			if e.slot == parts[1] {
				was, next, _, ok := socketRecolour(b, e)
				if !ok {
					return nil, "", errf(CodeInvalidValue, "The gems in %s already match its socket colours.", parts[1])
				}
				return recolourChange(b, e.id, next), "recolour " + parts[1] + " sockets " + was + " -> " + next, nil
			}
		}
		return nil, "", errf(CodeInvalidSlot, "Nothing equipped in %s.", parts[1])
	case "cluster":
		if len(parts) != 4 {
			return bad()
		}
		node, err := strconv.Atoi(parts[1])
		if err != nil {
			return bad()
		}
		for _, sock := range activeSpec(b).Sockets {
			if sock.NodeID != node {
				continue
			}
			for _, it := range b.Items.Items {
				if it.ID == sock.ItemID && strings.Contains(it.Text, notablePrefix+parts[2]) {
					ch, path := clusterSwap(b, int64(node), it.ID, strings.Split(it.Text, "\n"), parts[2], parts[3])
					note := "cluster jewel in " + parts[1] + ": " + parts[2] + " -> " + parts[3]
					if len(path) > 0 {
						note += fmt.Sprintf(", allocating %d nodes", len(path))
					}
					return ch, note, nil
				}
			}
		}
		return nil, "", errf(CodeInvalidValue, "No cluster jewel with %s in socket %s.", parts[2], parts[1])
	case "node":
		n, ok := ints(1)
		if !ok || len(n) == 0 {
			return bad()
		}
		p := planPaths(b, specTreeVersion(b)).path(int64(n[0]))
		if p == nil {
			return nil, "", errf(CodeInvalidValue, "Node %d has no path from the allocated tree.", n[0])
		}
		return func(c *pob.PathOfBuilding) {
			c.AllocateNodes(p)
			if len(n) > 1 {
				c.SetMasteryEffect(int64(n[0]), int64(n[1]))
			}
		}, "allocate node " + parts[1], nil
	case "links":
		n, ok := ints(2)
		if !ok || len(n) != 2 {
			return bad()
		}
		return linksScanChange(b, parts[1], n[0], n[1])
	case "linkup":
		if len(parts) != 4 {
			return bad()
		}
		group, err := strconv.Atoi(parts[2])
		if err != nil {
			return bad()
		}
		return linkUpScanChange(b, parts[1], group, parts[3])
	case "gemswap":
		n, err := strconv.Atoi(parts[1])
		if len(parts) != 4 || err != nil {
			return bad()
		}
		at, err := strconv.Atoi(parts[2])
		if err != nil {
			return bad()
		}
		return gemSwapScanChange(b, n, at, parts[3])
	case "gemlevel", "gemquality", "gemremove":
		n, ok := ints(1)
		groups := *groupsOf(b)
		if !ok || len(n) < 2 || n[0] < 1 || n[0] > len(groups) || n[1] < 1 || n[1] > len(groups[n[0]-1].Gems) {
			return bad()
		}
		gem := groups[n[0]-1].Gems[n[1]-1]
		switch {
		case parts[0] == "gemremove":
			return gemRemoveChange(n[0], n[1]-1), "remove " + gem.NameSpec, nil
		case parts[0] == "gemlevel" && len(n) == 4:
			return gemLevelChange(n[0], n[1]-1, n[2], n[3]), fmt.Sprintf("set %s to %d/%d", gem.NameSpec, n[2], n[3]), nil
		case len(n) != 3:
			return bad()
		case parts[0] == "gemlevel":
			return gemLevelChange(n[0], n[1]-1, n[2], gem.Quality), "set " + gem.NameSpec + " level " + parts[3], nil
		default:
			return gemLevelChange(n[0], n[1]-1, gem.Level, n[2]), "set " + gem.NameSpec + " quality " + parts[3], nil
		}
	}
	return bad()
}

func (s *Server) scanApply(in scanIn) (any, *ToolError) {
	return s.runEdit(in.Handle, in.DryRun, in.Watch, 1, func(ec *editCtx, _ int) *ToolError {
		ch, note, e := scanChange(ec.b, in.Apply)
		if e != nil {
			return e
		}
		ch(ec.b)
		ec.note(note)
		return nil
	})
}
