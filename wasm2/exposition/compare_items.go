package exposition

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/Vilsol/go-pob/calculator2"
)

const colorPositive = "^x33FF77"

var (
	tradeNumberRe = regexp.MustCompile(`-?\d+\.?\d*`)
	tradeRangeRe  = regexp.MustCompile(`(-?\d+\.?\d*) to (-?\d+\.?\d*)`)
)

//pob2:src/Classes/TradeHelpers.lua:82-85
func tradeModLineTemplate(line string) string {
	return tradeNumberRe.ReplaceAllString(line, "#")
}

// The value is nil (ok false) where the line has no number.
//
//pob2:src/Classes/TradeHelpers.lua:91-99
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
}

// compareModLists is the enchant, scourge, implicit, explicit and crucible lists; a PoE2 item has no scourge or crucible
// lines.
func compareModLists(item *calculator2.Item) [][]*calculator2.ModLine {
	return [][]*calculator2.ModLine{item.EnchantModLines, item.ImplicitModLines, item.ExplicitModLines}
}

//pob2:src/Classes/TradeHelpers.lua:377-392
func tradeBuildModMap(item *calculator2.Item) map[string]tradeModEntry {
	modMap := map[string]tradeModEntry{}
	if item == nil {
		return modMap
	}
	for _, modList := range compareModLists(item) {
		for _, modLine := range modList {
			if !item.CheckModLineVariant(modLine) {
				continue
			}
			if _, ok := formatModLine(item, modLine); ok {
				v, _ := tradeModLineValue(modLine.Line)
				modMap[tradeModLineTemplate(modLine.Line)] = tradeModEntry{line: modLine.Line, value: v}
			}
		}
	}
	return modMap
}

// luaD is string.format("%d", v).
func luaD(v float64) int64 { return int64(v) }

// itemExpanded is DrawItemExpanded's lines for item, its mod lines marked against otherModMap; level is the primary
// build's character level.
//
//pob2:src/Classes/CompareTab.lua:3494-3659
func itemExpanded(item *calculator2.Item, otherModMap map[string]tradeModEntry, level int) []string {
	if item == nil {
		return []string{"^8(empty)"}
	}
	var out []string
	emit := func(format string, a ...any) { out = append(out, fmt.Sprintf(format, a...)) }
	rarityColor, ok := rarityColors[item.Rarity]
	if !ok {
		rarityColor = "^7"
	}
	emit("%s%s", rarityColor, item.Name)
	if base := item.Base; base != nil {
		switch {
		case base.Weapon != nil:
			if weaponData := item.WeaponData[1]; weaponData != nil {
				for _, k := range []struct{ key, label string }{{"PhysicalDPS", "Phys DPS"}, {"ElementalDPS", "Ele DPS"}, {"ChaosDPS", "Chaos DPS"}, {"TotalDPS", "Total DPS"}} {
					if v, ok := weaponData[k.key]; ok {
						emit("^x7F7F7F%s: "+colorMagic+"%.1f", k.label, v)
					}
				}
				emit("^x7F7F7FCrit: "+colorMagic+"%.2f%%", weaponData["CritChance"])
				emit("^x7F7F7FAPS: "+colorMagic+"%.2f", weaponData["AttackRate"])
			}
		case base.Armour != nil:
			if armourData := item.ArmourData; armourData != nil {
				for _, k := range []struct{ key, label string }{{"Armour", "Armour"}, {"Evasion", "Evasion"}, {"EnergyShield", "ES"}, {"Ward", "Runic Ward"}} {
					if v := item.ArmourDataValue(k.key, level); v > 0 {
						emit("^x7F7F7F%s: "+colorMagic+"%d", k.label, luaD(v))
					}
				}
				if v := armourData["BlockChance"]; v > 0 {
					emit("^x7F7F7FBlock: "+colorMagic+"%d%%", luaD(v))
				}
			}
		case base.Flask != nil:
			if flaskData := item.FlaskData; flaskData != nil {
				life, hasLife := flaskData["lifeTotal"]
				mana, hasMana := flaskData["manaTotal"]
				if hasLife {
					emit("^x7F7F7FLife: "+colorMagic+"%d ^x7F7F7F(%.1fs)", luaD(life), flaskData["duration"])
				}
				if hasMana {
					emit("^x7F7F7FMana: "+colorMagic+"%d ^x7F7F7F(%.1fs)", luaD(mana), flaskData["duration"])
				}
				if d, ok := flaskData["duration"]; !hasLife && !hasMana && ok {
					emit("^x7F7F7FDuration: "+colorMagic+"%.2fs", d)
				}
				used, okUsed := flaskData["chargesUsed"]
				charges, okMax := flaskData["chargesMax"]
				if okUsed && okMax {
					emit("^x7F7F7FCharges: "+colorMagic+"%d/%d", luaD(used), luaD(charges))
				}
				// PoB walks buffModLines with pairs(); it is a list
				for _, modLine := range item.BuffModLines {
					color := colorMagic
					if modLine.Extra != "" {
						color = colorUnsupported
					}
					emit("%s%s", color, modLine.Line)
				}
			}
		}
		if item.Quality != nil && *item.Quality > 0 && base.Weapon == nil && base.Armour == nil && base.Flask == nil {
			emit("^x7F7F7FQuality: "+colorMagic+"+%d%%", luaD(*item.Quality))
		}
	}
	for _, modList := range compareModLists(item) {
		for _, modLine := range modList {
			if !item.CheckModLineVariant(modLine) {
				continue
			}
			formatted, ok := formatModLine(item, modLine)
			if !ok {
				continue
			}
			if otherModMap != nil {
				if otherEntry, found := otherModMap[tradeModLineTemplate(modLine.Line)]; !found {
					formatted = colorPositive + "+ " + formatted
				} else if otherEntry.line != modLine.Line {
					myVal, _ := tradeModLineValue(modLine.Line)
					switch {
					case myVal > otherEntry.value:
						formatted = colorPositive + "> " + formatted
					case myVal < otherEntry.value:
						formatted = colorNegative + "< " + formatted
					}
				}
			}
			out = append(out, formatted)
		}
	}
	if item.Corrupted {
		out = append(out, colorNegative+"Corrupted")
	}
	if item.Mirrored {
		out = append(out, colorNegative+"Mirrored")
	}
	return out
}

// CompareItemCardsView is the Items view's expanded mode for one slot: this build's item, then from's.
type CompareItemCardsView struct {
	A []string
	B []string
}

// CompareItemCards is the expanded mode's two items of slotName, each marked against the other.
//
//pob2:block:src/Classes/CompareTab.lua:3803-3807
func (c *Calculator) CompareItemCards(from *Calculator, slotName string) (*CompareItemCardsView, error) {
	if from == nil {
		return nil, errNoCompare
	}
	primary, err := c.comparePowerSide()
	if err != nil {
		return nil, err
	}
	compare, err := from.comparePowerSide()
	if err != nil {
		return nil, err
	}
	slots, jewels := calculator2.CompareItemSlots(primary, compare)
	for _, r := range append(slots, jewels...) {
		if r.Slot == slotName {
			pModMap, cModMap := tradeBuildModMap(r.PItem), tradeBuildModMap(r.CItem)
			level := c.env.Player.Level
			return &CompareItemCardsView{A: itemExpanded(r.PItem, cModMap, level), B: itemExpanded(r.CItem, pModMap, level)}, nil
		}
	}
	return &CompareItemCardsView{A: []string{}, B: []string{}}, nil
}
