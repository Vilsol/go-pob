package exposition

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

//crystalline:exports
func ItemExports(r bind.Registry) {
	r.Type(ParsedItem{}, bind.Plain())
	r.Type(ItemModLine{}, bind.Plain())
	r.Type(ItemProperty{}, bind.Plain())
	r.Type(calculator.ItemTooltip{}, bind.Plain())
	r.Type(calculator.TooltipLine{}, bind.Plain())
	r.Type(calculator.ItemTooltipStats{}, bind.Plain())
	r.Type(ItemRef{}, bind.Plain())
	r.Type(SlotCompare{}, bind.Plain())
	r.Type(FlaskCompare{}, bind.Plain())
	r.Type(AddedItem{}, bind.Plain())
	r.Type(EquippedSlot{}, bind.Plain())
	r.Type(SlotItems{}, bind.Plain())

	r.Func(ParseItem, bind.AsPromise())
	r.Func(EquipItem, bind.AsPromise())
	r.Func(EquippedItems, bind.AsPromise())
	r.Func(SetActiveItemSet, bind.AsPromise())
	r.Func(EquipItemID, bind.AsPromise())
	r.Func(SlotItemLists, bind.AsPromise())
}

type ItemModLine struct {
	Line string
	Kind string
}

type ItemProperty struct {
	Label string
	Value string
}

type ParsedItem struct {
	Name        string
	Title       string
	BaseName    string
	Type        string
	Rarity      string
	Quality     int
	ItemLevel   int
	Corrupted   bool
	Mirrored    bool
	Split       bool
	Fractured   bool
	Synthesised bool
	Influences  []string
	Sockets     string
	Properties  []ItemProperty
	Level       int
	Str         int
	Dex         int
	Int         int
	Enchants    []ItemModLine
	Scourge     []ItemModLine
	Implicits   []ItemModLine
	Explicits   []ItemModLine
	Flask       bool
	FitSlots    []string
	Tooltip     calculator.ItemTooltip
	ArmourData  map[string]float64
}

var errInvalidItem = fmt.Errorf("invalid item: no base type resolved")

var fitSlots = []string{"Weapon 1", "Weapon 2", "Helmet", "Body Armour", "Gloves", "Boots", "Amulet", "Ring 1", "Ring 2", "Belt",
	"Flask 1", "Flask 2", "Flask 3", "Flask 4", "Flask 5"}

func parseItem(text string) (*calculator.Item, error) {
	if strings.TrimSpace(text) == "" {
		return nil, errInvalidItem
	}
	item := calculator.ParseItem(text)
	if item.Base == nil {
		return nil, errInvalidItem
	}
	return item, nil
}

func buildItem(build *pob.PathOfBuilding, text string) (*calculator.Item, error) {
	if build != nil && strings.TrimSpace(text) != "" {
		for _, it := range build.Items.Items {
			if it.Text == text {
				if item := calculator.PeekItem(text); item.Base != nil {
					return item, nil
				}
				return nil, errInvalidItem
			}
		}
	}
	return parseItem(text)
}

func ParseItem(build *pob.PathOfBuilding, text string, stats calculator.ItemTooltipStats) (ParsedItem, error) {
	item, err := buildItem(build, text)
	if err != nil {
		return ParsedItem{}, err
	}
	return parsedItem(build, item, stats), nil
}

func parsedItem(build *pob.PathOfBuilding, item *calculator.Item, stats calculator.ItemTooltipStats) ParsedItem {
	p := ParsedItem{
		Name:        item.Name,
		Title:       item.Title,
		BaseName:    item.BaseName,
		Type:        item.Type,
		Rarity:      string(item.Rarity),
		Corrupted:   item.Corrupted,
		Mirrored:    item.Mirrored,
		Split:       item.Split,
		Fractured:   item.Fractured,
		Synthesised: item.Synthesised,
		Influences:  influences(item.Influence),
		Sockets:     socketLine(item.Sockets),
		Properties:  properties(item),
		Level:       int(item.Requirements["level"]),
		Str:         int(item.Requirements["strMod"]),
		Dex:         int(item.Requirements["dexMod"]),
		Int:         int(item.Requirements["intMod"]),
		Enchants:    modLines(item, item.EnchantModLines),
		Scourge:     modLines(item, item.ScourgeModLines),
		Implicits:   modLines(item, item.ImplicitModLines),
		Explicits:   append(modLines(item, item.ExplicitModLines), modLines(item, item.CrucibleModLines)...),
		Flask:       item.Base.Flask != nil || item.Base.Tincture != nil,
	}
	if item.Base.Armour != nil {
		p.ArmourData = item.ArmourData
	}
	ctx := calculator.ItemTooltipContext{ShowFlavourText: showFlavourText, Stats: stats}
	if build != nil {
		ctx.CharacterLevel = build.Build.Level
		ctx.ClassName = string(build.Build.ClassName)
	}
	p.Tooltip = calculator.AddItemTooltip(item, ctx)
	appendItemModDebug(&p.Tooltip, item)
	p.Tooltip.Children = calculator.GrantedSkillTooltips(p.Tooltip, ctx)
	if item.Quality != nil {
		p.Quality = *item.Quality
	}
	if item.ItemLevel != nil {
		p.ItemLevel = *item.ItemLevel
	}
	if build != nil {
		for _, s := range fitSlots {
			if calculator.IsItemValidForSlot(build, item, s) {
				p.FitSlots = append(p.FitSlots, s)
			}
		}
	}
	return p
}

func influences(i calculator.ItemInfluence) []string {
	var out []string
	for _, f := range []struct {
		on   bool
		name string
	}{
		{i.Shaper, "Shaper"}, {i.Elder, "Elder"}, {i.Adjudicator, "Warlord"}, {i.Basilisk, "Hunter"},
		{i.Crusader, "Crusader"}, {i.Eyrie, "Redeemer"}, {i.Cleansing, "Searing Exarch"}, {i.Tangle, "Eater of Worlds"},
	} {
		if f.on {
			out = append(out, f.name)
		}
	}
	return out
}

func socketLine(sockets []calculator.ItemSocket) string {
	var b strings.Builder
	for i, s := range sockets {
		if i > 0 {
			if s.Group == sockets[i-1].Group {
				b.WriteByte('-')
			} else {
				b.WriteByte(' ')
			}
		}
		b.WriteString(s.Color)
	}
	return b.String()
}

//pob:block:src/Classes/ItemsTab.lua:4625-4640
func modLines(item *calculator.Item, lines []calculator.ModLine) []ItemModLine {
	var out []ItemModLine
	for _, ml := range lines {
		if !item.ItemModLineVisible(ml) {
			continue
		}
		if line, kind, ok := calculator.FormatModLine(ml); ok {
			out = append(out, ItemModLine{Line: line, Kind: string(kind)})
		}
	}
	return out
}

//pob:block:src/Classes/ItemsTab.lua:4449-4505
func properties(item *calculator.Item) []ItemProperty {
	var out []ItemProperty
	add := func(label, format string, args ...any) {
		out = append(out, ItemProperty{Label: label, Value: fmt.Sprintf(format, args...)})
	}
	quality := 0
	if item.Quality != nil {
		quality = *item.Quality
	}
	switch {
	case item.Base.Weapon != nil:
		wd := item.WeaponData[1]
		label := item.Base.Type
		if info := data.WeaponTypes[data.ItemClassName(item.Base.Type)]; info != nil && info.Label != "" {
			label = info.Label
		} else if item.Base.SubType != "" {
			label = item.Base.SubType
		}
		add("", "%s", label)
		if quality > 0 {
			add("Quality", "+%d%%", quality)
		}
		if wd == nil {
			return out
		}
		types := 0
		if _, ok := wd["PhysicalDPS"]; ok {
			add("Physical Damage", "%d-%d (%.1f DPS)", int(wd["PhysicalMin"]), int(wd["PhysicalMax"]), wd["PhysicalDPS"])
			types++
		}
		if _, ok := wd["ElementalDPS"]; ok {
			var parts []string
			for _, t := range []string{"Fire", "Cold", "Lightning"} {
				if _, ok := wd[t+"DPS"]; ok {
					parts = append(parts, fmt.Sprintf("%d-%d", int(wd[t+"Min"]), int(wd[t+"Max"])))
				}
			}
			add("Elemental Damage", "%s", strings.Join(parts, ", "))
			add("Elemental DPS", "%.1f", wd["ElementalDPS"])
			types++
		}
		if _, ok := wd["ChaosDPS"]; ok {
			add("Chaos Damage", "%d-%d (%.1f DPS)", int(wd["ChaosMin"]), int(wd["ChaosMax"]), wd["ChaosDPS"])
			types++
		}
		if types > 1 {
			add("Total DPS", "%.1f", wd["TotalDPS"])
		}
		add("Critical Strike Chance", "%.2f%%", wd["CritChance"])
		add("Attacks per Second", "%.2f", wd["AttackRate"])
		if wd["range"] < 120 {
			add("Weapon Range", "%.1f metres", wd["range"]/10)
		}
	case item.Base.Armour != nil:
		ad := item.ArmourData
		if quality > 0 {
			add("Quality", "+%d%%", quality)
		}
		if item.Base.Armour.BlockChance != 0 && ad["BlockChance"] > 0 {
			add("Chance to Block", "%d%%", int(ad["BlockChance"]))
		}
		for _, p := range []struct{ key, label string }{
			{"Armour", "Armour"}, {"Evasion", "Evasion Rating"}, {"EnergyShield", "Energy Shield"}, {"Ward", "Ward"},
		} {
			if ad[p.key] > 0 {
				add(p.label, "%d", int(ad[p.key]))
			}
		}
	}
	return out
}

type ItemRef struct {
	Name   string
	Rarity string
}

type SlotCompare struct {
	SlotName  string
	Label     string
	Replacing *ItemRef
	Output    StatOutputs
}

func (m *MiscCalculator) CompareItem(text string) ([]SlotCompare, error) {
	item, err := parseItem(text)
	if err != nil {
		return nil, err
	}
	var out []SlotCompare
	for _, c := range m.calc.CompareItemSlots(item) {
		s := SlotCompare{SlotName: c.SlotName, Label: c.Label, Output: statOutputs(c.Env)}
		if c.Replacing != nil {
			s.Replacing = &ItemRef{Name: c.Replacing.Name, Rarity: string(c.Replacing.Rarity)}
		}
		out = append(out, s)
	}
	return out, nil
}

func (m *MiscCalculator) CalcUnequip(slotName string) StatOutputs {
	return statOutputs(m.calc.Calc(&calculator.CalcOverride{RepSlotName: slotName}))
}

type FlaskCompare struct {
	Active    bool
	Tincture  bool
	Output    StatOutputs
	Effective []string
}

//pob:block:src/Classes/ItemsTab.lua:5003-5010,src/Classes/ItemsTab.lua:5039-5046
func (m *MiscCalculator) CalcToggleFlask(text string) (FlaskCompare, error) {
	item, err := parseItem(text)
	if err != nil {
		return FlaskCompare{}, err
	}
	override := &calculator.CalcOverride{ToggleFlask: item}
	tincture := item.Base.Tincture != nil
	if tincture {
		override = &calculator.CalcOverride{ToggleTincture: item}
	} else if item.Base.Flask == nil {
		return FlaskCompare{}, fmt.Errorf("%s is not a flask or tincture", item.Name)
	}
	effective := m.calc.EffectiveFlaskStats(item)
	if tincture {
		effective = m.calc.EffectiveTinctureStats(item)
	}
	return FlaskCompare{Active: m.calc.FlaskActive(item), Tincture: tincture, Output: statOutputs(m.calc.Calc(override)), Effective: effective}, nil
}

type AddedItem struct {
	ID       int
	SlotName string
}

func (m *MiscCalculator) AddItem(text string) (AddedItem, error) {
	if _, err := parseItem(text); err != nil {
		return AddedItem{}, err
	}
	id, slot := m.calc.AddItem(text)
	return AddedItem{ID: id, SlotName: slot}, nil
}

func (m *MiscCalculator) ReplaceItem(id int, text string) error {
	if _, err := parseItem(text); err != nil {
		return err
	}
	if !m.calc.ReplaceItem(id, text) {
		return fmt.Errorf("the build has no item %d", id)
	}
	return nil
}

func EquipItem(build *pob.PathOfBuilding, text string, slotName string) (int, error) {
	item, err := parseItem(text)
	if err != nil {
		return 0, err
	}
	if !calculator.IsItemValidForSlot(build, item, slotName) {
		return 0, fmt.Errorf("%s does not fit %s", item.Name, slotName)
	}
	return calculator.EquipItem(build, text, slotName), nil
}

type EquippedSlot struct {
	SlotName string
	ItemID   int
	Name     string
	Rarity   string
	Flask    bool
	Active   bool
	Text     string
}

func EquippedItems(build *pob.PathOfBuilding) []EquippedSlot {
	if len(build.Items.ItemSets) == 0 {
		return nil
	}
	set := build.Items.ItemSets[0]
	for _, s := range build.Items.ItemSets {
		if s.ID == fmt.Sprint(build.Items.ActiveItemSet) {
			set = s
		}
	}
	texts := make(map[int]string, len(build.Items.Items))
	for _, it := range build.Items.Items {
		texts[it.ID] = it.Text
	}
	var out []EquippedSlot
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
			e := EquippedSlot{SlotName: s.Name, ItemID: s.ItemID, Name: item.Name, Rarity: string(item.Rarity), Active: s.Active, Text: text}
			e.Flask = item.Base != nil && (item.Base.Flask != nil || item.Base.Tincture != nil)
			out = append(out, e)
		}
	}
	return out
}

func SetActiveItemSet(build *pob.PathOfBuilding, itemSetID int) {
	build.SetActiveItemSet(itemSetID)
	calculator.PopulateSlots(build)
}

func EquipItemID(build *pob.PathOfBuilding, slotName string, itemID int) (bool, error) {
	if itemID != 0 && !slices.Contains(calculator.SlotItemIDs(build, slotName), itemID) {
		return false, fmt.Errorf("item %d does not fit %s", itemID, slotName)
	}
	return calculator.EquipItemID(build, slotName, itemID), nil
}

type SlotItems struct {
	SlotName string
	ItemIDs  []int
}

func SlotItemLists(build *pob.PathOfBuilding) []SlotItems {
	names := calculator.OrderedItemSlotNames()
	out := make([]SlotItems, 0, len(names))
	for n, ids := range calculator.SlotItemIDsFor(build, names) {
		out = append(out, SlotItems{SlotName: names[n], ItemIDs: ids})
	}
	return out
}
