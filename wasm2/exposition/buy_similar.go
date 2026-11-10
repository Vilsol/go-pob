package exposition

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Vilsol/go-pob/calculator2"
	"github.com/Vilsol/go-pob/data/poe2"
)

const errNoItem = itemsError("no item in that slot")

// BuySimilarView is the Buy Similar popup for the item this build equips in a slot, as the Compare tab opens it for
// the compared build's item: whether it searches by unique name, the name it shows, the Category label, the base the
// base box names and the defence rows.
type BuySimilarView struct {
	Name     string
	BaseName string
	Category string
	Defences []calculator2.BuySimilarDefence
	Mods     []BuySimilarMod
	Listed   []string
	Unique   bool
}

// BuySimilarMod is a mod row: its source type, its label lines with their colour codes, whether its box can be
// checked, whether it has Min and Max boxes and the Min box's text. There are none when go-pob-data does not publish
// the trade site's stats list.
type BuySimilarMod struct {
	Type      string
	Min       string
	Lines     []string
	CanSearch bool
	HasBoxes  bool
}

var (
	buySimilarColorRe     = regexp.MustCompile(`\^x[0-9A-Fa-f]{6}`)
	buySimilarShortColRe  = regexp.MustCompile(`\^[0-9A-Fa-f]`)
	buySimilarTimeLostTag = " Passive Skills in Radius also grant"
)

// buySimilarModEntries is the popup's mod entries for item; none when the trade data cannot be read.
func buySimilarModEntries(item *calculator2.Item) []*calculator2.BuySimilarModEntry {
	if _, err := poe2.TradeStats(); err != nil {
		return nil
	}
	if _, err := poe2.StatDescriptions(); err != nil {
		return nil
	}
	return calculator2.BuySimilarAddModEntries(item, func(ml *calculator2.ModLine) (string, bool) {
		return formatModLine(item, ml)
	})
}

// buySimilarMods is the popup's mod rows as openPopup lays them out.
//
//pob2:block:src/Classes/CompareBuySimilar.lua:459-516
func buySimilarMods(modEntries []*calculator2.BuySimilarModEntry) []BuySimilarMod {
	mods := make([]BuySimilarMod, 0, len(modEntries))
	for _, entry := range modEntries {
		canSearch := len(entry.TradeIDs) > 0
		displayTexts := make([]string, len(entry.FormattedLines))
		for index, displayText := range entry.FormattedLines {
			// shorten time-lost jewel affix labels to fit better
			displayText = strings.ReplaceAll(displayText, buySimilarTimeLostTag, ":")
			colorCodeLength := buySimilarColorRe.FindString(displayText)
			if colorCodeLength == "" {
				colorCodeLength = buySimilarShortColRe.ReplaceAllString(displayText, "")
			}
			if !canSearch {
				// strip color codes and replace with gray
				displayText = "^8" + buySimilarShortColRe.ReplaceAllString(buySimilarColorRe.ReplaceAllString(displayText, ""), "")
			}
			if len(displayText) > len(colorCodeLength)+62 {
				displayText = displayText[:len(colorCodeLength)+54] + "..."
			}
			displayTexts[index] = displayText
		}
		mod := BuySimilarMod{Type: entry.Type, Lines: displayTexts, CanSearch: canSearch}
		// when the trade site has a dropdown for the value, we opt to disable
		// the inputs as they are numeric
		if !entry.IsOption && entry.Value != nil {
			mod.HasBoxes = true
			if *entry.Value != 0 {
				mod.Min = fmt.Sprintf("%.14g", *entry.Value)
			}
		}
		mods = append(mods, mod)
	}
	return mods
}

func (c *Calculator) buySimilarItem(slotName string) (*calculator2.Item, error) {
	tab, err := c.itemsTab()
	if err != nil {
		return nil, err
	}
	slot := tab.Slots[slotName]
	if slot == nil || tab.Items[slot.SelItemID] == nil {
		return nil, errNoItem
	}
	return tab.Items[slot.SelItemID], nil
}

// BuySimilar is the popup for the item in slotName.
//
//pob2:block:src/Classes/CompareBuySimilar.lua:284-329
//pob2:block:src/Classes/CompareBuySimilar.lua:421-434
//pob2:block:src/Classes/CompareBuySimilar.lua:459-516
func (c *Calculator) BuySimilar(slotName string) (*BuySimilarView, error) {
	item, err := c.buySimilarItem(slotName)
	if err != nil {
		return nil, err
	}
	listed := make([]string, len(calculator2.BuySimilarListed))
	for i, l := range calculator2.BuySimilarListed {
		listed[i] = l.Label
	}
	return &BuySimilarView{
		Unique:   item.Rarity == "UNIQUE" || item.Rarity == "RELIC",
		Name:     item.Name,
		BaseName: item.BaseName,
		Category: calculator2.TradeCategoryLabel(item),
		Defences: calculator2.BuySimilarDefences(item),
		Mods:     buySimilarMods(buySimilarModEntries(item)),
		Listed:   listed,
	}, nil
}

// BuySimilarURL is the Open URL button's trade search for the item in slotName with the popup's selections.
func (c *Calculator) BuySimilarURL(slotName string, o calculator2.BuySimilarOptions) (string, error) {
	item, err := c.buySimilarItem(slotName)
	if err != nil {
		return "", err
	}
	return calculator2.BuySimilarURL(item, slotName, buySimilarModEntries(item), o), nil
}
