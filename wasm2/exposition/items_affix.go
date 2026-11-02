package exposition

import "github.com/Vilsol/go-pob/calculator2"

// AffixEntryView is one entry of an affix drop-down: a mod, or a series of its tiers (ModList, lowest first).
type AffixEntryView struct {
	Label     string
	ModList   []string
	HaveRange bool
}

// AffixControlView is one of the editor's affix drop-downs; List[0] is "None" and SelIndex is 1-based.
type AffixControlView struct {
	Type        string
	List        []AffixEntryView
	SelIndex    int
	SliderShown bool
	SliderVal   float64
}

// DisplayItemAffixes is the editor's affix drop-downs, prefixes first; none for an item that is not crafted.
//
//pob2:block:src/Classes/ItemsTab.lua:2232-2246
func (c *Calculator) DisplayItemAffixes() ([]AffixControlView, error) {
	tab, err := c.itemsTab()
	if err != nil {
		return nil, err
	}
	out := []AffixControlView{}
	for _, ctl := range tab.AffixControls() {
		typ := "Prefix"
		if ctl.OutputTable == "suffixes" {
			typ = "Suffix"
		}
		v := AffixControlView{Type: typ, SelIndex: ctl.SelIndex, SliderShown: ctl.SliderShown, SliderVal: ctl.SliderVal, List: []AffixEntryView{}}
		for _, e := range ctl.List {
			modList := e.ModList
			if modList == nil {
				modList = []string{}
			}
			v.List = append(v.List, AffixEntryView{Label: e.Label, ModList: modList, HaveRange: e.HaveRange})
		}
		out = append(out, v)
	}
	return out, nil
}

// SelectDisplayItemAffix picks entry (1-based) of affix drop-down #control, with its roll slider at slider.
func (c *Calculator) SelectDisplayItemAffix(control, entry int, slider float64) error {
	return c.displayEdit(func(tab *calculator2.ItemsTab) error { return tab.SelectDisplayItemAffix(control, entry, slider) })
}

// SetDisplayItemAffixRange moves affix drop-down #control's roll slider to val.
func (c *Calculator) SetDisplayItemAffixRange(control int, val float64) error {
	return c.displayEdit(func(tab *calculator2.ItemsTab) error { return tab.SetDisplayItemAffixRange(control, val) })
}

// DisplayItemAffixRoll is affix drop-down #control's roll slider tooltip at val.
func (c *Calculator) DisplayItemAffixRoll(control int, val float64) ([]string, error) {
	tab, err := c.itemsTab()
	if err != nil {
		return nil, err
	}
	return tab.AffixRollTooltip(control, val)
}

// RangeLineView is one of the editor's modifier range lines and its roll.
type RangeLineView struct {
	Label string
	Range float64
}

// DisplayItemRanges is the editor's modifier range lines, and its quality when it shows one (-1 when not).
func (c *Calculator) DisplayItemRanges() ([]RangeLineView, int, error) {
	tab, err := c.itemsTab()
	if err != nil {
		return nil, -1, err
	}
	out := []RangeLineView{}
	for _, l := range tab.DisplayItemRangeLines() {
		out = append(out, RangeLineView{Label: l.Label, Range: l.Range})
	}
	quality := -1
	if tab.DisplayItemQualityShown() {
		quality = int(*tab.DisplayItem.Quality)
	}
	return out, quality, nil
}

// SetDisplayItemRangeLine moves range line #line's (1-based) slider to val.
func (c *Calculator) SetDisplayItemRangeLine(line int, val float64) error {
	return c.displayEdit(func(tab *calculator2.ItemsTab) error { return tab.SetDisplayItemRangeLine(line, val) })
}

// SetDisplayItemQuality is the editor's quality box.
func (c *Calculator) SetDisplayItemQuality(quality int) error {
	return c.displayEdit(func(tab *calculator2.ItemsTab) error { return tab.SetDisplayItemQuality(quality) })
}
