package exposition

import (
	"regexp"
	"strings"

	"github.com/Vilsol/go-pob/calculator2"
	"github.com/Vilsol/go-pob/data/poe2skills"
)

// SkillGemView is one gem of a socket group as the skills tab lists it.
type SkillGemView struct {
	// ID is the gem's data id, "" for a skill no gem grants.
	ID      string
	Name    string
	Level   int
	Quality float64
	Count   float64
	Enabled bool
	Support bool
	// Color is the gem's colour code (colorCodes.STRENGTH, DEXTERITY, INTELLIGENCE or NORMAL).
	Color string
	// Generated is a gem the group's source grants: not editable or removable.
	Generated bool
	// CountShown and CorruptShown are whether the gem slot shows its Count box and Corrupt dropdown.
	CountShown   bool
	CorruptShown bool
	// CorruptIndex is the Corrupt dropdown's entry, 1-based.
	CorruptIndex int
	// Error is the gem slot's error message.
	Error string
}

// SkillGroupView is one socket group of the skills tab: its display label, where it comes from, and its gems.
type SkillGroupView struct {
	// Label is the display label; UserLabel the group's own.
	Label            string
	UserLabel        string
	GroupCount       float64
	Slot             string
	Source           string
	Enabled          bool
	IncludeInFullDPS bool
	Set1             bool
	Set2             bool
	// WeaponSetLocked is a group whose source item's weapon set decides its sets.
	WeaponSetLocked bool
	// Set1Valid and Set2Valid are whether the group can be used in each weapon set; Set1Enabled and Set2Enabled
	// whether its box can be changed; ForcedBoth a group that must be in both.
	Set1Valid   bool
	Set2Valid   bool
	Set1Enabled bool
	Set2Enabled bool
	ForcedBoth  bool
	Gems        []SkillGemView
}

// Global.lua:7-78 (PoB-PoE2)
const (
	colorNormal       = "^xC8C8C8"
	colorMagic        = "^x8888FF"
	colorCrafted      = "^xB8DAF1"
	colorEnchanted    = "^xB8DAF1"
	colorCustom       = "^x5CF0BB"
	colorUnsupported  = "^xF05050"
	colorDisabled     = "^x7F7F7F"
	colorFractured    = "^xA29160"
	colorMutated      = "^xAE2E3B"
	colorNegative     = "^xDD0022"
	colorStrength     = "^xE05030"
	colorDexterity    = "^x70FF70"
	colorIntelligence = "^x7070FF"
)

var rarityColors = map[string]string{
	"NORMAL": colorNormal,
	"MAGIC":  colorMagic,
	"RARE":   "^xFFFF77",
	"UNIQUE": "^xAF6025",
	"RELIC":  "^x60C060",
}

//pob2:block:src/Classes/SkillsTab.lua:1351-1361
func gemColor(gem *calculator2.GemInstance) string {
	grantedEffect := gem.GrantedEffect
	if grantedEffect == nil && gem.GemData != nil {
		grantedEffect = gem.GemData.GrantedEffect
	}
	if grantedEffect == nil {
		return colorNormal
	}
	switch grantedEffect.Color {
	case 1:
		return colorStrength
	case 2:
		return colorDexterity
	case 3:
		return colorIntelligence
	}
	return colorNormal
}

// gemSlotShown is whether one of the gem's effects is an active skill the sidebar lists, as the Count box and the
// Corrupt dropdown ask.
//
//pob2:block:src/Classes/SkillsTab.lua:1090-1099
func gemSlotShown(gem *calculator2.GemInstance) bool {
	var list []*poe2skills.GrantedEffect
	if gem.GemData != nil {
		list = gem.GemData.GrantedEffectList
	} else {
		list = []*poe2skills.GrantedEffect{gem.GrantedEffect}
	}
	for i, grantedEffect := range list {
		if grantedEffect == nil {
			continue
		}
		hide := grantedEffect.Overlay != nil && grantedEffect.Overlay.HideFromSideBar
		if !grantedEffect.Support && !hide && (!grantedEffect.HasGlobalEffect || i < 2 && (i == 0 && gem.EnableGlobal1 || i == 1 && gem.EnableGlobal2)) {
			return true
		}
	}
	return false
}

// SkillGroups is the active skill set's socket groups as the last calculation loaded them.
//
//pob2:block:src/Classes/SkillListControl.lua:78-96
func (c *Calculator) SkillGroups() ([]SkillGroupView, error) {
	if c.env == nil {
		return nil, notCalculatedError{}
	}
	labels := calculator2.SocketGroupViews(c.env)
	out := make([]SkillGroupView, len(c.env.SocketGroupList))
	for i, group := range c.env.SocketGroupList {
		view := SkillGroupView{
			UserLabel:        group.Label,
			GroupCount:       group.GroupCount,
			Slot:             group.Slot,
			Source:           group.Source,
			Enabled:          group.Enabled,
			IncludeInFullDPS: group.IncludeInFullDPS,
			Set1:             group.Set1,
			Set2:             group.Set2,
			Gems:             make([]SkillGemView, 0, len(group.GemList)),
		}
		if i < len(labels) {
			view.Label = labels[i].Label
		}
		//pob2:block:src/Classes/SkillsTab.lua:1393-1396
		if (group.SourceItem != nil || group.Source == "Default Attack") && group.Slot != "" && c.items != nil {
			slot := c.items.Slots[group.Slot]
			view.WeaponSetLocked = slot != nil && slot.WeaponSet != 0
		}
		if !group.HasGroupCount {
			view.GroupCount = 1
		}
		valid, enabled, forcedBoth := calculator2.SocketGroupWeaponSets(c.env, i+1)
		view.Set1Valid, view.Set2Valid, view.Set1Enabled, view.Set2Enabled, view.ForcedBoth = valid[0], valid[1], enabled[0], enabled[1], forcedBoth
		for gemIndex, gem := range group.GemList {
			grantedEffect := gem.GrantedEffect
			if grantedEffect == nil && gem.GemData != nil {
				grantedEffect = gem.GemData.GrantedEffect
			}
			name := gem.NameSpec
			if gem.GemData != nil && gem.GemData.Data != nil {
				name = gem.GemData.Data.Name
			} else if grantedEffect != nil {
				name = grantedEffect.Name
			}
			generated := gemIndex == 0 && (group.Source != "" || group.SourceItem != nil || group.SourceNode != nil)
			shown := !generated && gemSlotShown(gem)
			id := ""
			if gem.GemData != nil {
				id = gem.GemData.Data.ID
			}
			view.Gems = append(view.Gems, SkillGemView{
				ID:           id,
				Generated:    generated,
				CountShown:   shown,
				CorruptShown: gemSlotShown(gem) && !gem.FromItem && !gem.FromTree && (grantedEffect == nil || grantedEffect.Overlay == nil || !grantedEffect.Overlay.FromItem && !grantedEffect.Overlay.FromTree),
				CorruptIndex: calculator2.CorruptIndex(gem),
				Error:        calculator2.GemErrMsg(gem),
				Name:         name,
				Level:        gem.Level,
				Quality:      gem.Quality,
				Count:        gem.Count,
				Enabled:      gem.Enabled,
				Support:      grantedEffect != nil && grantedEffect.Support,
				Color:        gemColor(gem),
			})
		}
		out[i] = view
	}
	return out, nil
}

// ItemView is one equipped item: the slot it is in and the lines its tooltip names it and lists its mods with.
type ItemView struct {
	Slot string
	ID   int
	// Name is the tooltip's title lines, coloured by rarity.
	Name   []string
	Rarity string
	// Lines are the tooltip's mod lines, one group per enchant, rune, implicit and explicit list that has lines,
	// then the corrupted labels.
	Lines [][]string
}

// ItemsView is the active item set: which weapon set it uses, and the item in each shown slot that holds one.
type ItemsView struct {
	UseSecondWeaponSet bool
	Items              []ItemView
}

var (
	zeroLineStartRe = regexp.MustCompile(`^\+?0%? `)
	zeroLineMidRe   = regexp.MustCompile(` \+?0%? `)
	zeroLineToRe    = regexp.MustCompile(`0 to [1-9]`)
	zeroLinePctToRe = regexp.MustCompile(`0% to \d+%`)
	baseSuffixRe    = regexp.MustCompile(` \(.+\)`)
)

//pob2:src/Modules/ItemTools.lua:170-172
func isZeroValueLine(line string) bool {
	return zeroLineStartRe.MatchString(line) ||
		(zeroLineMidRe.MatchString(line) && !zeroLineToRe.MatchString(line) && !zeroLinePctToRe.MatchString(line)) ||
		strings.Contains(line, " 0-0 ") || strings.Contains(line, " 0 to 0 ")
}

// formatModLine is itemLib.formatModLine outside dbMode.
//
//pob2:src/Modules/ItemTools.lua:438-458
func formatModLine(item *calculator2.Item, modLine *calculator2.ModLine) (string, bool) {
	line := item.RangedLine(modLine)
	if isZeroValueLine(line) {
		return "", false
	}
	if modLine.Disabled {
		return colorDisabled + line, true
	}
	var colorCode string
	switch {
	case modLine.Extra != "":
		colorCode = colorUnsupported
	case modLine.Fractured:
		colorCode = colorFractured
	case modLine.Crafted:
		colorCode = colorCrafted
	case modLine.Enchant:
		colorCode = colorEnchanted
	case modLine.Mutated:
		colorCode = colorMutated
	case modLine.Custom && !modLine.Desecrated:
		colorCode = colorCustom
	default:
		colorCode = colorMagic
	}
	return colorCode + line, true
}

// itemTooltip is ItemsTab:AddItemTooltip's title and mod lines; the jewel radius scale, granted passives
// and cluster jewel nodes are not ported.
//
//pob2:block:src/Classes/ItemsTab.lua:3720-3729
//pob2:block:src/Classes/ItemsTab.lua:4039-4141
func itemTooltip(item *calculator2.Item) ItemView {
	view := ItemView{ID: item.ID, Rarity: item.Rarity}
	rarityCode := rarityColors[item.Rarity]
	switch {
	case item.Base == nil || item.BaseName == "":
		name := item.Title
		if name == "" {
			name = item.Name
		}
		if name == "" {
			name = "Unknown Item"
		}
		view.Name = []string{rarityCode + name}
		return view
	case item.Title != "":
		view.Name = []string{rarityCode + item.Title, rarityCode + baseSuffixRe.ReplaceAllString(item.BaseName, "")}
	default:
		view.Name = []string{rarityCode + item.NamePrefix + baseSuffixRe.ReplaceAllString(item.BaseName, "") + item.NameSuffix}
	}
	for _, modList := range [][]*calculator2.ModLine{item.EnchantModLines, item.RuneModLines, item.ImplicitModLines, item.ExplicitModLines} {
		if len(modList) == 0 {
			continue
		}
		var lines []string
		for _, modLine := range modList {
			variantCount := item.GetModLineVariantCount(modLine)
			if variantCount == 0 {
				continue
			}
			if formatted, ok := formatModLine(item, modLine); ok {
				for range variantCount {
					lines = append(lines, formatted)
				}
			}
		}
		view.Lines = append(view.Lines, lines)
	}
	var labels []string
	if item.Mirrored {
		labels = append(labels, colorNegative+"Mirrored")
	}
	if item.Sanctified {
		labels = append(labels, colorFractured+"Sanctified")
	}
	if item.DoubleCorrupted {
		labels = append(labels, colorNegative+"Twice Corrupted")
	} else if item.Corrupted {
		labels = append(labels, colorNegative+"Corrupted")
	}
	if len(labels) > 0 {
		view.Lines = append(view.Lines, labels)
	}
	return view
}

// EquippedItems is the item in every slot of the active item set that holds one, in orderedSlots order: the
// weapon slots of the weapon set in use, item jewel sockets under their item, and tree jewel sockets on
// allocated nodes.
//
//pob2:block:src/Classes/ItemsTab.lua:210-262
//pob2:block:src/Classes/ItemsTab.lua:1739-1760
func (c *Calculator) EquippedItems() (*ItemsView, error) {
	if c.env == nil || c.items == nil {
		return nil, notCalculatedError{}
	}
	out := &ItemsView{UseSecondWeaponSet: c.items.UseSecondWeaponSet, Items: []ItemView{}}
	shown := func(slot *calculator2.ItemSlot) bool {
		if slot.NodeID != 0 {
			return c.env.AllocNodes[slot.NodeID] != nil
		}
		if parent := slot.ParentSlot; parent != nil {
			// ItemSlotControl.lua:136-144: a socket past the parent item's jewelSocketCount is inactive
			parentItem := c.items.Items[parent.SelItemID]
			if parentItem == nil || slot.SlotNum > parentItem.JewelSocketCount {
				return false
			}
			slot = parent
		}
		if slot.SlotName == "Ring 3" && !c.env.ModDB.Flag(nil, "AdditionalRingSlot") {
			return false
		}
		switch slot.WeaponSet {
		case 1:
			return !c.items.UseSecondWeaponSet
		case 2:
			return c.items.UseSecondWeaponSet
		}
		return true
	}
	for _, slot := range c.items.OrderedSlots {
		item := c.items.Items[slot.SelItemID]
		if slot.SelItemID == 0 || item == nil || !shown(slot) {
			continue
		}
		view := itemTooltip(item)
		view.Slot = slot.SlotName
		out.Items = append(out.Items, view)
	}
	return out, nil
}
