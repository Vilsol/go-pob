package calculator

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob-data/poe"

	raw2 "github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/utils"
)

const (
	colorGem     = "^x1AA29B"
	colorWarning = "^xFF9922"
	colorNormal  = "^xC8C8C8"
)

type GemDisplay struct {
	Level          int
	Quality        int
	SocketQuality  int
	ItemQuality    int
	SupportQuality int
	GlobalQuality  int
}

//pob:block:src/Classes/GemTooltip.lua:16-21
var gemReservationMap = []struct{ name, res string }{
	{"manaReservationFlat", "Mana"},
	{"manaReservationPercent", "ManaPercent"},
	{"lifeReservationFlat", "Life"},
	{"lifeReservationPercent", "LifePercent"},
}

//pob:block:src/Data/Costs.lua:4-89
var gemCostTypes = []struct {
	resource, resourceString string
	divisor                  float64
}{
	{"Mana", "{0} Mana", 1},
	{"Life", "{0} Life", 1},
	{"ES", "{0} Energy Shield", 1},
	{"Rage", "{0} Rage", 1},
	{"ManaPercent", "{0}% Mana", 1},
	{"LifePercent", "{0}% Life", 1},
	{"UnreservedManaPercent", "[DNT] {0}% Unreserved Mana ", 1},
	{"ManaPerMinute", "{0} Mana per second", 60},
	{"LifePerMinute", "{0} Life per second", 60},
	{"ManaPercentPerMinute", "{0}% Mana per second", 60},
	{"LifePercentPerMinute", "{0}% Life per second", 60},
	{"ESPerMinute", "{0} Energy Shield per second", 60},
	{"ESPercentPerMinute", "{0}% Energy Shield per second", 60},
	{"Soul", "{0} Souls Per Use", 1},
}

func gemCostResourceString(res string) string {
	for _, c := range gemCostTypes {
		if c.resource == res {
			return c.resourceString
		}
	}
	return ""
}

func luaD(v float64) string {
	return strconv.FormatFloat(math.Trunc(v), 'f', 0, 64)
}

func gemLevelReservation(level *raw2.CalculatedLevel, name string) *float64 {
	switch name {
	case "manaReservationFlat":
		return level.ManaReservationFlat
	case "manaReservationPercent":
		return level.ManaReservationPercent
	case "lifeReservationFlat":
		return level.LifeReservationFlat
	case "lifeReservationPercent":
		return level.LifeReservationPercent
	}
	return nil
}

func gemReservationText(level *raw2.CalculatedLevel) string {
	var reservation []string
	for _, r := range gemReservationMap {
		if v := gemLevelReservation(level, r.name); v != nil {
			reservation = append(reservation, strings.ReplaceAll(gemCostResourceString(r.res), "{0}", luaD(*v)))
		}
	}
	return strings.Join(reservation, ", ")
}

type gemTooltipEffect struct {
	id      string
	text    raw2.SkillText
	support bool
	levels  map[int]*raw2.CalculatedLevel
	wrapped *GrantedEffect
}

func newGemTooltipEffect(id string) *gemTooltipEffect {
	effect := poe.GrantedEffectByID(id)
	if effect == nil {
		return nil
	}
	text, _ := raw2.SkillTextFor(id)
	return &gemTooltipEffect{
		id:      id,
		text:    text,
		support: effect.IsSupport,
		levels:  raw2.GetCalculatedGrantedEffect(effect).GetCalculatedLevels(),
		wrapped: &GrantedEffect{Raw: effect},
	}
}

func (g *gemTooltipEffect) level(level int) *raw2.CalculatedLevel {
	if l := g.levels[level]; l != nil {
		return l
	}
	return &raw2.CalculatedLevel{}
}

func gemTooltipRow(gem pob.Gem) (*poe.SkillGem, raw2.PoBGem, bool) {
	gemData, _ := resolveGemInstance(gem)
	if row, ok := pobGemForVariant(gemData, gem.SkillID); ok {
		return gemData, row, true
	}
	if row, ok := raw2.Gems[gem.GemID]; ok {
		return gemData, row, true
	}
	row, ok := raw2.Gems[strings.Replace(gem.GemID, "/SupportGem", "/SkillGemSupport", 1)]
	return gemData, row, ok
}

func gemTooltipSecondary(gemData *poe.SkillGem, row raw2.PoBGem, gem pob.Gem) *poe.GrantedEffect {
	if gemData == nil {
		if row.SecondaryGrantedEffectID == "" {
			return nil
		}
		return poe.GrantedEffectByID(row.SecondaryGrantedEffectID)
	}
	return secondaryGrantedEffectOf(gemData, gem.SkillID, gem.VariantID)
}

type gemTooltipInstance struct {
	gem     pob.Gem
	row     raw2.PoBGem
	text    raw2.GemText
	display GemDisplay
}

//pob:src/Classes/GemTooltip.lua:30-205
//nolint:gocyclo // Mirrors the Lua branch chain.
func addCommonGemInfo(t *ItemTooltip, ctx ItemTooltipContext, gi *gemTooltipInstance, grantedEffect *gemTooltipEffect, addLevel, addReq bool, mergeStatsFrom *gemTooltipEffect) {
	display := gi.display
	grantedEffectLevel := grantedEffect.level(display.Level)
	if addLevel {
		bonus := ""
		switch {
		case display.Level > gi.gem.Level:
			bonus = " (" + colorMagic + "+" + strconv.Itoa(display.Level-gi.gem.Level) + "^7)"
		case display.Level < gi.gem.Level:
			bonus = " (" + colorWarning + "-" + strconv.Itoa(gi.gem.Level-display.Level) + "^7)"
		}
		maxed := ""
		if gi.row.NaturalMaxLevel != 0 && gi.gem.Level >= gi.row.NaturalMaxLevel {
			maxed = " (Max)"
		}
		t.addLine(fmt.Sprintf("^x7F7F7FLevel: ^7%d%s%s", gi.gem.Level, bonus, maxed))
	}
	if grantedEffect.support {
		if grantedEffectLevel.ManaMultiplier != nil {
			t.addLine("^x7F7F7FCost & Reservation Multiplier: ^7" + luaD(*grantedEffectLevel.ManaMultiplier+100) + "%")
		}
		if reservation := gemReservationText(grantedEffectLevel); reservation != "" {
			t.addLine("^x7F7F7FReservation Override: ^7" + reservation)
		}
		if grantedEffectLevel.Cooldown != nil {
			line := fmt.Sprintf("^x7F7F7FCooldown Time: ^7%.2f sec", *grantedEffectLevel.Cooldown)
			if grantedEffectLevel.StoredUses != nil && *grantedEffectLevel.StoredUses > 1 {
				line += " (" + luaD(*grantedEffectLevel.StoredUses) + " uses)"
			}
			t.addLine(line)
		}
	} else {
		if reservation := gemReservationText(grantedEffectLevel); reservation != "" {
			t.addLine("^x7F7F7FReservation: ^7" + reservation)
		}
		var cost []string
		for _, res := range gemCostTypes {
			if v, ok := grantedEffectLevel.Cost[res.resource]; ok {
				cost = append(cost, strings.ReplaceAll(res.resourceString, "{0}", strconv.FormatFloat(luaRound(float64(v)/res.divisor, 2), 'g', 6, 64)))
			}
		}
		if len(cost) > 0 {
			t.addLine("^x7F7F7FCost: ^7" + strings.Join(cost, ", "))
		}
		if grantedEffectLevel.Cooldown != nil {
			line := fmt.Sprintf("^x7F7F7FCooldown Time: ^7%.2f sec", *grantedEffectLevel.Cooldown)
			if grantedEffectLevel.StoredUses != nil && *grantedEffectLevel.StoredUses > 1 {
				line += " (" + luaD(*grantedEffectLevel.StoredUses) + " uses)"
			}
			t.addLine(line)
		}
		if grantedEffectLevel.SkillUseStorage != nil {
			t.addLine("^x7F7F7FCan Store ^7" + luaD(*grantedEffectLevel.SkillUseStorage) + " ^x7F7F7FUse (" + luaD(*grantedEffectLevel.SkillUseStorage*float64(grantedEffectLevel.Cost["Soul"])) + " Souls)")
		}
		if grantedEffectLevel.SoulPreventionDuration != nil {
			t.addLine("^x7F7F7FSoul Gain Prevention: ^7" + luaD(*grantedEffectLevel.SoulPreventionDuration) + " sec")
		}
		if gi.row.Tags["attack"] {
			if grantedEffectLevel.AttackSpeedMultiplier != nil {
				t.addLine("^x7F7F7FAttack Speed: ^7" + luaD(*grantedEffectLevel.AttackSpeedMultiplier+100) + "% of base")
			}
			if grantedEffectLevel.AttackTime != nil {
				t.addLine(fmt.Sprintf("^x7F7F7FAttack Time: ^7%.2f sec", *grantedEffectLevel.AttackTime/1000))
			}
			if grantedEffectLevel.BaseMultiplier != nil {
				t.addLine("^x7F7F7FAttack Damage: ^7" + strconv.FormatFloat(luaRound(*grantedEffectLevel.BaseMultiplier, 4)*100, 'g', 6, 64) + "% of base")
			}
		} else {
			if grantedEffect.text.CastTime > 0 {
				t.addLine(fmt.Sprintf("^x7F7F7FCast Time: ^7%.2f sec", grantedEffect.text.CastTime))
			} else {
				t.addLine("^x7F7F7FCast Time: ^7Instant")
			}
		}
		if grantedEffectLevel.CritChance != nil {
			t.addLine(fmt.Sprintf("^x7F7F7FCritical Strike Chance: ^7%.2f%%", *grantedEffectLevel.CritChance))
		}
		if grantedEffectLevel.DamageEffectiveness != nil {
			t.addLine("^x7F7F7FEffectiveness of Added Damage: ^7" + luaD(luaRound(*grantedEffectLevel.DamageEffectiveness, 4)*100) + "%")
		}
	}
	if addReq {
		if gi.gem.Quality > 0 {
			t.addLine(fmt.Sprintf("^x7F7F7FQuality: +%s%d%%", colorMagic, gi.gem.Quality))
		}
		for _, q := range []struct {
			n    int
			from string
		}{{display.ItemQuality, "Item"}, {display.SupportQuality, "Support"}, {display.GlobalQuality, "Global Modifiers"}, {display.SocketQuality, "Socket Colour"}} {
			if q.n > 0 {
				t.addLine(colorMagic + fmt.Sprintf("+%d%% Quality from %s", q.n, q.from))
			}
		}
	}
	t.addSeparator()
	if addReq {
		reqLevel := 1.0
		if l := grantedEffect.levels[gi.gem.Level]; l != nil {
			reqLevel = float64(l.LevelRequirement)
		}
		reqStr := getGemStatRequirement(reqLevel, grantedEffect.support, float64(gi.row.ReqStr))
		reqDex := getGemStatRequirement(reqLevel, grantedEffect.support, float64(gi.row.ReqDex))
		reqInt := getGemStatRequirement(reqLevel, grantedEffect.support, float64(gi.row.ReqInt))
		addRequirementsToTooltip(t, ctx, reqLevel, reqStr, reqDex, reqInt, reqStr, reqDex, reqInt)
	}
	if grantedEffect.text.Description != "" {
		t.addLine(colorGem + grantedEffect.text.Description)
	}
	t.addSeparator()
	instance := &GemEffect{Level: display.Level, Quality: display.Quality, QualityID: gi.gem.QualityID}
	stats := CalcBuildSkillInstanceStats(instance, grantedEffect.wrapped)
	if mergeStatsFrom != nil {
		for stat, val := range CalcBuildSkillInstanceStats(instance, mergeStatsFrom.wrapped) {
			stats[stat] += val
		}
	}
	described := DescribeStats(stats, grantedEffect.text.Scope, false)
	calculated := raw2.GetCalculatedGrantedEffect(grantedEffect.wrapped.Raw)
	for _, line := range described.Lines {
		if calculated.GetStatMapEntry(described.LineMap[line]) != nil {
			t.addLine(colorMagic + line)
		} else {
			t.addLine(colorUnsupported + line + notSupportedTooltipText)
		}
	}
}

//pob:src/Classes/GemTooltip.lua:213-259
func AddGemTooltip(t *ItemTooltip, ctx ItemTooltipContext, gem pob.Gem, display *GemDisplay, skipRequirements bool) bool {
	gemData, row, ok := gemTooltipRow(gem)
	if !ok {
		return false
	}
	return addGemTooltip(t, ctx, gem, gemData, row, display, skipRequirements)
}

//pob:block:src/Classes/ItemsTab.lua:4624-4630
func AddSkillTooltip(t *ItemTooltip, ctx ItemTooltipContext, skillID string, level int) bool {
	return addGemTooltip(t, ctx, pob.Gem{SkillID: skillID, Level: level, Enabled: true}, nil, raw2.PoBGem{GrantedEffectID: skillID, Tags: map[string]bool{}}, nil, true)
}

func addGemTooltip(t *ItemTooltip, ctx ItemTooltipContext, gem pob.Gem, gemData *poe.SkillGem, row raw2.PoBGem, display *GemDisplay, skipRequirements bool) bool {
	primary := newGemTooltipEffect(row.GrantedEffectID)
	if primary == nil {
		return false
	}
	gi := &gemTooltipInstance{gem: gem, row: row}
	gi.text, _ = raw2.GemTextFor(row.GrantedEffectID)
	if display != nil {
		gi.display = *display
	} else {
		gi.display = GemDisplay{Level: gem.Level, Quality: gem.Quality}
	}
	var secondary *gemTooltipEffect
	if effect := gemTooltipSecondary(gemData, row, gem); effect != nil {
		secondary = newGemTooltipEffect(effect.ID)
	}

	if secondary != nil && (!secondary.support || gi.text.SecondaryEffectName != "") {
		grantedEffect, grantedEffectSecondary := primary, secondary
		t.addLine(colorGem + grantedEffect.text.Name)
		t.addSeparator()
		t.addLine("^x7F7F7F" + gi.text.TagString)
		addCommonGemInfo(t, ctx, gi, grantedEffect, true, !skipRequirements, nil)
		t.addSeparator()
		t.addLine(colorGem + utils.Ternary(gi.text.SecondaryEffectName != "", gi.text.SecondaryEffectName, grantedEffectSecondary.text.Name))
		t.addSeparator()
		addCommonGemInfo(t, ctx, gi, grantedEffectSecondary, false, false, nil)
	} else {
		grantedEffect := primary
		t.addLine(colorGem + grantedEffect.text.Name)
		t.addSeparator()
		if grantedEffect.text.Legacy {
			t.addLine(colorWarning + "Legacy Gem")
			t.addLine(colorWarning + "Gem only exists in Standard League")
			t.addSeparator()
		}
		if gi.text.TagString != "" {
			t.addLine("^x7F7F7F" + gi.text.TagString)
		}
		var merge *gemTooltipEffect
		if secondary != nil && secondary.support {
			merge = secondary
		}
		addCommonGemInfo(t, ctx, gi, grantedEffect, true, !skipRequirements, merge)
	}
	if len(primary.text.FlavourText) > 0 && ctx.ShowFlavourText {
		t.addSeparator()
		for _, line := range primary.text.FlavourText {
			t.addLine(colorUnique + line)
		}
	}
	return true
}

//pob:src/Classes/SkillsTab.lua:895-936
func GemQualityTooltip(gem pob.Gem) []TooltipLine {
	t := &ItemTooltip{}
	gemData, row, ok := gemTooltipRow(gem)
	if !ok {
		return nil
	}
	addQualityLines := func(grantedEffect *gemTooltipEffect) bool {
		qualityList := grantedEffect.text.QualityStats
		if len(qualityList) == 0 {
			return false
		}
		t.addLine(colorGem + grantedEffect.text.Name)
		t.addLine(colorNormal + "At +20% Quality:")
		calculated := raw2.GetCalculatedGrantedEffect(grantedEffect.wrapped.Raw)
		for _, qual := range qualityList {
			stats := map[string]float64{qual.Stat: qual.Value * 20}
			for _, line := range DescribeStats(stats, grantedEffect.text.Scope, false).Lines {
				if calculated.GetStatMapEntry(qual.Stat) != nil {
					t.addLine(colorMagic + line)
				} else {
					t.addLine(colorUnsupported + line + notSupportedTooltipText)
				}
			}
		}
		return true
	}
	if primary := newGemTooltipEffect(row.GrantedEffectID); primary != nil && addQualityLines(primary) {
		t.addSeparator()
	}
	if effect := gemTooltipSecondary(gemData, row, gem); effect != nil {
		if secondary := newGemTooltipEffect(effect.ID); secondary != nil && addQualityLines(secondary) {
			t.addSeparator()
		}
	}
	return t.Lines
}

func (env *Environment) setDisplayEffect(group, gem int, effect *GemEffect) {
	if env.displayEffects == nil {
		env.displayEffects = map[[2]int]*GemEffect{}
	}
	env.displayEffects[[2]int{group, gem}] = effect
}

type GemDisplayRow struct {
	Group   int
	Gem     int
	Display GemDisplay
}

func (env *Environment) GemDisplays() []GemDisplayRow {
	if env == nil {
		return nil
	}
	rows := make([]GemDisplayRow, 0, len(env.displayEffects))
	for key, effect := range env.displayEffects {
		rows = append(rows, GemDisplayRow{Group: key[0], Gem: key[1], Display: GemDisplay{
			Level: effect.Level, Quality: effect.Quality, SocketQuality: effect.SocketQuality,
			ItemQuality: effect.ItemQuality, SupportQuality: effect.SupportQuality, GlobalQuality: effect.GlobalQuality,
		}})
	}
	slices.SortFunc(rows, func(a, b GemDisplayRow) int {
		if a.Group != b.Group {
			return a.Group - b.Group
		}
		return a.Gem - b.Gem
	})
	return rows
}

type GemSupportPair struct {
	Support int
	Active  int
}

//pob:block:src/Classes/GemSelectControl.lua:631-638
func (env *Environment) GemSupportPairs(group int) []GemSupportPair {
	if env == nil || env.Build == nil {
		return nil
	}
	skillSet := activeSkillSetIndex(env.Build.Skills)
	if skillSet < 0 || group < 0 || group >= len(env.Build.Skills.SkillSets[skillSet].Skills) {
		return nil
	}
	socketGroup := &env.Build.Skills.SkillSets[skillSet].Skills[group]
	gems := socketGroup.Gems
	index := func(inst *pob.Gem) int {
		if inst == nil {
			return -1
		}
		for j := range gems {
			if gems[j] == *inst {
				return j
			}
		}
		return -1
	}
	displaySkillList, _ := pob.DisplaySkillListOf(socketGroup).([]*ActiveSkill)
	seen := map[GemSupportPair]bool{}
	var out []GemSupportPair
	for _, activeSkill := range displaySkillList {
		if activeSkill == nil || activeSkill.ActiveEffect == nil {
			continue
		}
		a := index(activeSkill.ActiveEffect.SrcInstance)
		if a < 0 || !gems[a].Enabled || activeSkill.ActiveEffect.GemData == nil || gemIsSupport(activeSkill.ActiveEffect.GemData) {
			continue
		}
		for _, support := range activeSkill.SupportList {
			if support == nil || support.GrantedEffect == nil || !support.GrantedEffect.IsSupport() {
				continue
			}
			if !support.IsSupporting[activeSkill.ActiveEffect.SrcInstance] {
				continue
			}
			si := index(support.SrcInstance)
			p := GemSupportPair{Support: si, Active: a}
			if si < 0 || si == a || !gems[si].Enabled || seen[p] {
				continue
			}
			seen[p] = true
			out = append(out, p)
		}
	}
	slices.SortFunc(out, func(a, b GemSupportPair) int {
		if a.Support != b.Support {
			return a.Support - b.Support
		}
		return a.Active - b.Active
	})
	return out
}
