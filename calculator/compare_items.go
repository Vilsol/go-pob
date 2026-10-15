package calculator

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"

	"github.com/Vilsol/go-pob/pob"
)

var (
	tradeNumberRe = regexp.MustCompile(`-?\d+\.?\d*`)
	tradeRangeRe  = regexp.MustCompile(`(-?\d+\.?\d*) to (-?\d+\.?\d*)`)
)

//pob:src/Classes/TradeHelpers.lua:82-85
func tradeModLineTemplate(line string) string {
	return tradeNumberRe.ReplaceAllString(line, "#")
}

//pob:src/Classes/TradeHelpers.lua:91-99
func tradeModLineValue(line string) (float64, bool) {
	if m := tradeRangeRe.FindStringSubmatch(line); m != nil {
		low, _ := strconv.ParseFloat(m[1], 64)
		high, _ := strconv.ParseFloat(m[2], 64)
		return (low + high) / 2, true
	}
	if m := tradeNumberRe.FindString(line); m != "" {
		v, err := strconv.ParseFloat(m, 64)
		return v, err == nil
	}
	return 0, false
}

type tradeModEntry struct {
	line  string
	value float64
	ok    bool
}

func (i *Item) compareModLists() [][]ModLine {
	return [][]ModLine{i.EnchantModLines, i.ScourgeModLines, i.ImplicitModLines, i.ExplicitModLines, i.CrucibleModLines}
}

//pob:src/Classes/TradeHelpers.lua:401-416
func tradeBuildModMap(item *Item) map[string]tradeModEntry {
	out := map[string]tradeModEntry{}
	if item == nil {
		return out
	}
	for _, list := range item.compareModLists() {
		for _, ml := range list {
			if !item.checkModLineVariant(ml) {
				continue
			}
			if _, ok := formattedModLine(ml, false); ok {
				v, ok := tradeModLineValue(ml.Line)
				out[tradeModLineTemplate(ml.Line)] = tradeModEntry{line: ml.Line, value: v, ok: ok}
			}
		}
	}
	return out
}

//pob:src/Classes/CompareTab.lua:3495-3666
func CompareItemExpanded(raw string, other string) []string {
	if raw == "" {
		return []string{"^8(empty)"}
	}
	item := ParseItem(raw)
	var otherItem *Item
	if other != "" {
		otherItem = ParseItem(other)
	}
	otherModMap := tradeBuildModMap(otherItem)
	var out []string
	emit := func(format string, a ...any) { out = append(out, fmt.Sprintf(format, a...)) }

	color, ok := rarityColorCodes[item.Rarity]
	if !ok {
		color = "^7"
	}
	emit("%s%s", color, item.Name)
	if base := item.Base; base != nil {
		switch {
		case base.Weapon != nil:
			if wd := item.WeaponData[1]; wd != nil {
				for _, k := range []struct{ key, label string }{{"PhysicalDPS", "Phys DPS"}, {"ElementalDPS", "Ele DPS"}, {"ChaosDPS", "Chaos DPS"}, {"TotalDPS", "Total DPS"}} {
					if v, ok := wd[k.key]; ok {
						emit("^x7F7F7F%s: "+colorMagic+"%.1f", k.label, v)
					}
				}
				emit("^x7F7F7FCrit: "+colorMagic+"%.2f%%", wd["CritChance"])
				emit("^x7F7F7FAPS: "+colorMagic+"%.2f", wd["AttackRate"])
			}
		case base.Armour != nil:
			if ad := item.ArmourData; ad != nil {
				for _, k := range []struct{ key, label, unit string }{{"Armour", "Armour", ""}, {"Evasion", "Evasion", ""}, {"EnergyShield", "ES", ""}, {"Ward", "Ward", ""}, {"BlockChance", "Block", "%%"}} {
					if v := ad[k.key]; v > 0 {
						emit("^x7F7F7F%s: "+colorMagic+"%d"+k.unit, k.label, luaInt(v))
					}
				}
			}
		case base.Flask != nil:
			if fd := item.FlaskData; fd != nil {
				life, hasLife := fd["lifeTotal"]
				mana, hasMana := fd["manaTotal"]
				if hasLife {
					emit("^x7F7F7FLife: "+colorMagic+"%d ^x7F7F7F(%.1fs)", luaInt(life), fd["duration"])
				}
				if hasMana {
					emit("^x7F7F7FMana: "+colorMagic+"%d ^x7F7F7F(%.1fs)", luaInt(mana), fd["duration"])
				}
				if d, ok := fd["duration"]; !hasLife && !hasMana && ok {
					emit("^x7F7F7FDuration: "+colorMagic+"%.2fs", d)
				}
				used, okUsed := fd["chargesUsed"]
				charges, okMax := fd["chargesMax"]
				if okUsed && okMax {
					emit("^x7F7F7FCharges: "+colorMagic+"%d/%d", luaInt(used), luaInt(charges))
				}
				for _, line := range base.Flask.Buff {
					c := colorMagic
					if ParseMod(line, false).Extra != "" {
						c = colorUnsupported
					}
					emit("%s%s", c, line)
				}
			}
		}
		if item.Quality != nil && *item.Quality > 0 && base.Weapon == nil && base.Armour == nil && base.Flask == nil {
			emit("^x7F7F7FQuality: "+colorMagic+"+%d%%", *item.Quality)
		}
	}

	for _, list := range item.compareModLists() {
		for _, ml := range list {
			if !item.checkModLineVariant(ml) {
				continue
			}
			formatted, ok := formattedModLine(ml, false)
			if !ok {
				continue
			}
			if o, found := otherModMap[tradeModLineTemplate(ml.Line)]; !found {
				formatted = colorPositive + "+ " + formatted
			} else if o.line != ml.Line {
				mine, _ := tradeModLineValue(ml.Line)
				switch {
				case mine > o.value:
					formatted = colorPositive + "> " + formatted
				case mine < o.value:
					formatted = colorNegative + "< " + formatted
				}
			}
			out = append(out, formatted)
		}
	}
	if item.Corrupted {
		out = append(out, colorNegative+"Corrupted")
	}
	if item.Split {
		out = append(out, colorNegative+"Split")
	}
	if item.Mirrored {
		out = append(out, colorNegative+"Mirrored")
	}
	return out
}

type CompareJewelSlot struct {
	Label      string
	NodeID     int
	PText      string
	CText      string
	PAllocated bool
	CAllocated bool
}

//pob:src/Classes/CompareTab.lua:1377-1441
func CompareJewelSlots(primary, compare *pob.PathOfBuilding) []CompareJewelSlot {
	jewels := func(build *pob.PathOfBuilding) map[int]string {
		out := map[int]string{}
		if i := build.ActiveSpecIndex(); i >= 0 {
			for _, s := range build.Tree.Specs[i].Sockets {
				for _, it := range build.Items.Items {
					if s.ItemID > 0 && it.ID == s.ItemID {
						out[s.NodeID] = it.Text
					}
				}
			}
		}
		return out
	}
	pJewels, cJewels := jewels(primary), jewels(compare)
	pSockets, cSockets := ActiveTreeSockets(primary), ActiveTreeSockets(compare)
	ids := slices.Collect(maps.Keys(pJewels))
	for id := range cJewels {
		if _, ok := pJewels[id]; !ok {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	out := make([]CompareJewelSlot, 0, len(ids))
	for _, id := range ids {
		label := "Socket"
		if n := slices.Index(pSockets, id); n >= 0 {
			label = fmt.Sprintf("Socket #%d", n+1)
		}
		out = append(out, CompareJewelSlot{Label: label, NodeID: id, PText: pJewels[id], CText: cJewels[id],
			PAllocated: slices.Contains(pSockets, id), CAllocated: slices.Contains(cSockets, id)})
	}
	return out
}
