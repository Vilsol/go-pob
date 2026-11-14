package mcp

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

func slotNames() []string {
	var out []string
	for _, s := range calculator.OrderedItemSlotNames() {
		if !strings.Contains(s, "Swap") && !strings.Contains(s, "Abyssal") {
			out = append(out, s)
		}
	}
	return out
}

func resolveSlot(name string) (string, *ToolError) {
	for _, s := range calculator.OrderedItemSlotNames() {
		if norm(s) == norm(name) {
			return s, nil
		}
	}
	switch norm(name) {
	case "ring":
		return "Ring 1", nil
	case "weapon", "main hand", "mainhand":
		return "Weapon 1", nil
	case "offhand", "off hand", "shield", "quiver":
		return "Weapon 2", nil
	case "body", "chest", "body armor":
		return "Body Armour", nil
	case "helm":
		return "Helmet", nil
	}
	e := errf(CodeInvalidSlot, "No slot named %q.", name)
	e.ValidSlots = slotNames()
	return "", e
}

func validSlotsFor(b *pob.PathOfBuilding, item *calculator.Item) []string {
	var out []string
	for _, s := range slotNames() {
		if calculator.IsItemValidForSlot(b, item, s) {
			out = append(out, s)
		}
	}
	return out
}

func uniqueNames() []string {
	items := calculator.ItemDB(calculator.ItemDBUnique)
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Name)
	}
	return out
}

func resolveUnique(name string) (string, *ToolError) {
	items := calculator.ItemDB(calculator.ItemDBUnique)
	it, ok, hits := resolveName(name, items, func(i *calculator.Item) string { return i.Name }, nil)
	if ok {
		return it.Name, nil
	}
	if len(hits) > 0 {
		e := errf(CodeAmbiguous, "%q matches several uniques.", name)
		for _, h := range hits[:min(len(hits), 8)] {
			e.Candidates = append(e.Candidates, h.Name)
		}
		return "", e
	}
	return "", notFound("unique", name, suggest(name, uniqueNames(), 5), "Retry with one of did_you_mean, or search_data kind=unique.")
}

// equipText validates and equips an item text in slot (default: the item's primary slot); it returns the slot used.
func equipText(b *pob.PathOfBuilding, text, slot string) (string, *calculator.Item, *ToolError) {
	item := calculator.ParseItem(text)
	if item == nil || item.Base == nil {
		e := errf(CodeParseError, "The item text has no known base type.")
		if item != nil {
			e.Line = item.BaseName
		}
		return "", nil, e
	}
	if slot == "" {
		slot = item.PrimarySlot()
	} else {
		var e *ToolError
		if slot, e = resolveSlot(slot); e != nil {
			return "", nil, e
		}
	}
	if !calculator.IsItemValidForSlot(b, item, slot) {
		e := errf(CodeInvalidSlot, "%s cannot go in %s.", item.Name, slot)
		e.ValidSlots = validSlotsFor(b, item)
		return "", nil, e
	}
	calculator.EquipItem(b, text, slot)
	return slot, item, nil
}

func slotItemName(b *pob.PathOfBuilding, slot string) string {
	for _, e := range equippedItems(b) {
		if e.slot == slot {
			return e.name
		}
	}
	return ""
}

type itemOp struct {
	Op      string          `json:"op"`
	Slot    string          `json:"slot"`
	Name    string          `json:"name"`
	Base    string          `json:"base"`
	Affixes []string        `json:"affixes"`
	Sockets string          `json:"sockets"`
	Text    string          `json:"text"`
	Active  *bool           `json:"active"`
	Node    json.RawMessage `json:"node"`
}

type editItemsIn struct {
	Handle string   `json:"handle"`
	DryRun bool     `json:"dry_run"`
	Watch  []string `json:"watch"`
	Ops    []itemOp `json:"ops"`
}

var itemOps = []string{"equip_unique", "craft", "equip_text", "unequip", "set_flask_active", "socket_jewel"}

func (s *Server) editItems(args json.RawMessage) (any, *ToolError) {
	var in editItemsIn
	if e := decode(args, schemaEditItems, &in); e != nil {
		return nil, e
	}
	if e := checkOps(len(in.Ops), 10); e != nil {
		return nil, e
	}
	return s.runEdit(in.Handle, in.DryRun, in.Watch, len(in.Ops), func(ec *editCtx, i int) *ToolError {
		return applyItemOp(ec, in.Ops[i], in.DryRun)
	})
}

func applyItemOp(ec *editCtx, op itemOp, dryRun bool) *ToolError {
	b := ec.b
	equip := func(text, what string) *ToolError {
		prevSlot := op.Slot
		if prevSlot != "" {
			if s, e := resolveSlot(prevSlot); e == nil {
				prevSlot = s
			}
		}
		old := ""
		if prevSlot != "" {
			old = slotItemName(b, prevSlot)
		}
		slot, item, e := equipText(b, text, op.Slot)
		if e != nil {
			return e
		}
		if old == "" {
			old = slotItemName(b, slot)
		}
		msg := fmt.Sprintf("%s %s in %s", what, item.Name, slot)
		if old != "" && old != item.Name {
			msg += " (replaced " + old + ")"
		}
		ec.note(msg)
		if dryRun {
			ec.add("item", map[string]any{"name": item.Name, "text": text})
		}
		if bad := unsupportedLines(item); len(bad) > 0 {
			ec.add("unsupported_lines", bad)
		}
		return nil
	}
	switch op.Op {
	case "equip_unique":
		name, e := resolveUnique(op.Name)
		if e != nil {
			return e
		}
		text, _ := calculator.ItemDBText(calculator.ItemDBUnique, name)
		return equip(text, "equip_unique")
	case "craft":
		text, e := craftItem(op.Base, op.Affixes, op.Sockets)
		if e != nil {
			return e
		}
		return equip(text, "craft")
	case "equip_text":
		if strings.TrimSpace(op.Text) == "" {
			return invalidArg("equip_text needs text.", nil)
		}
		return equip(strings.ReplaceAll(op.Text, "\r\n", "\n"), "equip_text")
	case "unequip":
		slot, e := resolveSlot(op.Slot)
		if e != nil {
			return e
		}
		old := slotItemName(b, slot)
		b.SetSlotItem(slot, 0)
		calculator.PopulateSlots(b)
		ec.note(fmt.Sprintf("unequip %s (%s)", slot, old))
	case "set_flask_active":
		slot, e := resolveSlot(op.Slot)
		if e != nil {
			return e
		}
		active := op.Active == nil || *op.Active
		b.SetFlaskActive(slot, active)
		ec.note(fmt.Sprintf("set %s active=%v", slot, active))
	case "socket_jewel":
		v := specTreeVersion(b)
		n, e := resolveNode(b, v, op.Node, nil)
		if e != nil {
			return e
		}
		if n.Type != "jewel" {
			return errf(CodeInvalidValue, "%s (id %d) is not a jewel socket.", n.Name, n.ID)
		}
		if !slices.Contains(b.Build.PassiveNodes, n.ID) {
			e := errf(CodeNotReachable, "Jewel socket %d is not allocated.", n.ID)
			e.Hint = "Allocate it first with edit_tree op=allocate."
			return e
		}
		text := op.Text
		if text == "" {
			if op.Name == "" {
				return invalidArg("socket_jewel needs name (a unique jewel) or text.", nil)
			}
			name, e := resolveUnique(op.Name)
			if e != nil {
				return e
			}
			text, _ = calculator.ItemDBText(calculator.ItemDBUnique, name)
		}
		item := calculator.ParseItem(text)
		if item == nil || item.Base == nil || item.Base.Type != "Jewel" {
			return errf(CodeInvalidSlot, "Only jewels go in a jewel socket.")
		}
		id := b.AddItem(text)
		b.SocketJewel(int(n.ID), id)
		ec.note(fmt.Sprintf("socket %s in jewel socket %d", item.Name, n.ID))
	default:
		return invalidArg(fmt.Sprintf("Unknown op %q.", op.Op), itemOps)
	}
	return nil
}

func unsupportedLines(item *calculator.Item) []string {
	var out []string
	for _, ml := range item.ExplicitModLines {
		if !item.ItemModLineVisible(ml) {
			continue
		}
		if len(ml.ModList) == 0 || ml.Extra != "" {
			out = append(out, ml.Line)
		}
	}
	return out
}

func baseNames() []string {
	out := make([]string, 0, len(calculator.ItemBases()))
	for n := range calculator.ItemBases() {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func resolveBase(name string) (string, *ToolError) {
	if calculator.ItemBaseByName(name) != nil {
		return name, nil
	}
	list := baseNames()
	if n, ok, hits := resolveName(name, list, func(s string) string { return s }, nil); ok {
		return n, nil
	} else if len(hits) > 1 {
		e := errf(CodeAmbiguous, "%q matches several bases.", name)
		e.Candidates = toAny(hits[:min(len(hits), 8)])
		return "", e
	}
	return "", notFound("item base", name, suggest(name, list, 5), "Retry with one of did_you_mean, or search_data kind=base.")
}

type affixChoice struct {
	slot   int
	series int
	label  string
	typ    string
}

// craftItem builds a rare through PoB's crafting path: CraftItemText, then each affix at the top of its slider.
func craftItem(baseName string, affixes []string, sockets string) (string, *ToolError) {
	if baseName == "" {
		return "", invalidArg("craft needs base.", nil)
	}
	base, e := resolveBase(baseName)
	if e != nil {
		return "", e
	}
	text, ok := calculator.CraftItemText(2, "", base)
	if !ok {
		return "", errf(CodeInvalidValue, "%s cannot be crafted.", base)
	}
	if sockets != "" {
		text = setSockets(text, sockets)
	}
	used := map[int]bool{}
	var chosen []string
	for _, label := range affixes {
		ctl, ok := calculator.ItemAffixControls(text)
		if !ok {
			return "", errf(CodeAffixLimit, "%s takes no crafted affixes.", base)
		}
		var options []affixChoice
		for si, c := range ctl.Controls {
			for k, ser := range c.Series {
				if k == 0 {
					continue
				}
				options = append(options, affixChoice{slot: si, series: k, label: ser.Label, typ: c.Type})
			}
		}
		pick, ok, _ := resolveName(label, options, func(a affixChoice) string { return a.label }, nil)
		if !ok {
			e := errf(CodeAffixLimit, "%q is not an affix %s can roll.", label, base)
			e.Chosen = chosen
			e.AllowedBase = suggest(label, uniqStrings(options), 10)
			e.Hint = "Use search_data kind=affix base=<base> to list labels."
			return "", e
		}
		slot := -1
		for si, c := range ctl.Controls {
			if used[si] || c.Type != pick.typ {
				continue
			}
			for k, ser := range c.Series {
				if k > 0 && ser.Label == pick.label {
					slot, pick.series = si, k
					break
				}
			}
			if slot >= 0 {
				break
			}
		}
		if slot < 0 {
			e := errf(CodeAffixLimit, "No free %s slot left for %q (at most 3 prefixes and 3 suffixes).", strings.ToLower(pick.typ), label)
			e.Chosen = chosen
			return "", e
		}
		next, ok := calculator.AffixEditText(text, slot, pick.series, 1)
		if !ok {
			return "", errf(CodeAffixLimit, "%q could not be applied to %s.", label, base)
		}
		used[slot] = true
		text = next
		chosen = append(chosen, pick.typ+": "+pick.label)
	}
	return text, nil
}

func uniqStrings(options []affixChoice) []string {
	seen := map[string]bool{}
	var out []string
	for _, o := range options {
		if !seen[o.label] {
			seen[o.label] = true
			out = append(out, o.label)
		}
	}
	return out
}

func setSockets(text, sockets string) string {
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, "Sockets: ") {
			lines[i] = "Sockets: " + sockets
			return strings.Join(lines, "\n")
		}
	}
	for i, l := range lines {
		if strings.HasPrefix(l, "Quality: ") || strings.HasPrefix(l, "LevelReq: ") || strings.HasPrefix(l, "Implicits: ") {
			return strings.Join(slices.Insert(lines, i, "Sockets: "+sockets), "\n")
		}
	}
	return text + "\nSockets: " + sockets
}

type equipped struct {
	slot   string
	id     int
	name   string
	rarity string
	active bool
	text   string
}

func equippedItems(b *pob.PathOfBuilding) []equipped {
	if len(b.Items.ItemSets) == 0 {
		return nil
	}
	set := b.Items.ItemSets[0]
	for _, s := range b.Items.ItemSets {
		if s.ID == fmt.Sprint(b.Items.ActiveItemSet) {
			set = s
		}
	}
	texts := make(map[int]string, len(b.Items.Items))
	for _, it := range b.Items.Items {
		texts[it.ID] = it.Text
	}
	var out []equipped
	second := set.UseSecondWeaponSet != nil && *set.UseSecondWeaponSet
	for _, name := range calculator.OrderedItemSlotNames() {
		if strings.HasPrefix(name, "Weapon") && strings.Contains(name, "Swap") != second {
			continue
		}
		for _, s := range set.Slots {
			text, ok := texts[s.ItemID]
			if s.Name != name || !ok {
				continue
			}
			item := calculator.PeekItem(text)
			out = append(out, equipped{slot: s.Name, id: s.ItemID, name: item.Name, rarity: string(item.Rarity), active: s.Active, text: text})
		}
	}
	return out
}
