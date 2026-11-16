package mcp

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
)

// gemFlag is a gem the audit flags, with its 1-based group; Set names a skill set other than the active one, and
// AlsoGroups the set's other groups holding the same gem at the same quality.
type gemFlag struct {
	Set        string `json:"set,omitempty"`
	Group      int    `json:"group"`
	AlsoGroups []int  `json:"also_groups,omitempty"`
	Gem        string `json:"gem"`
	Level      int    `json:"level,omitempty"`
	Quality    int    `json:"quality,omitempty"`
	Note       string `json:"note"`
}

// mergeGemFlags folds a flag into an earlier one for the same set, gem, level, quality and note.
func mergeGemFlags(in []gemFlag) []gemFlag {
	var out []gemFlag
	for _, f := range in {
		i := slices.IndexFunc(out, func(o gemFlag) bool {
			return o.Set == f.Set && o.Gem == f.Gem && o.Level == f.Level && o.Quality == f.Quality && o.Note == f.Note
		})
		if i < 0 {
			out = append(out, f)
		} else if f.Group != out[i].Group && !slices.Contains(out[i].AlsoGroups, f.Group) {
			out[i].AlsoGroups = append(out[i].AlsoGroups, f.Group)
		}
	}
	return out
}

// eachSetGem calls fn for every socketed gem of every skill set, with set "" for the active one; skills items grant
// are left out.
func eachSetGem(b *pob.PathOfBuilding, fn func(set string, group int, gem pob.Gem)) {
	active := b.ActiveSkillSetIndex()
	for si, ss := range b.Skills.SkillSets {
		set := ""
		if si != active {
			set = fmt.Sprintf("set %d", ss.ID)
			if ss.Title != nil && *ss.Title != "" {
				set = *ss.Title
			}
		}
		for i, g := range ss.Skills {
			if g.Source != "" {
				continue
			}
			for _, gem := range g.Gems {
				fn(set, i+1, gem)
			}
		}
	}
}

// gemSkillID is the granted effect a saved gem loads as: by gem id, else skill id, else its name.
//
//pob:block:src/Classes/SkillsTab.lua:1169-1201
func gemSkillID(gem pob.Gem) string {
	if row, ok := gemVariantRow(gem); ok {
		return row.GrantedEffectID
	}
	if row, ok := raw.Gems[gem.GemID]; ok && gem.GemID != "" {
		return row.GrantedEffectID
	}
	if gem.SkillID != "" {
		return gem.SkillID
	}
	if row, ok := gemRowByName(gem.NameSpec); ok {
		return row.GrantedEffectID
	}
	return ""
}

// gemVariantRow is a transfigured gem's own row: PoB saves it under its base gem's id with the variant's id, or in
// older builds the variant's skill id, and its granted effect, parts and labels are the variant's.
//
//pob:block:src/Classes/SkillsTab.lua:466-480
func gemVariantRow(gem pob.Gem) (raw.PoBGem, bool) {
	if gem.VariantID != nil {
		row, ok := raw.GemVariants[*gem.VariantID]
		return row, ok
	}
	if gem.SkillID != "" && gem.GemID != "" && raw.GemGameIDForGrantedEffectID[gem.SkillID] == gem.GemID {
		row, ok := raw.GemVariants[gem.SkillID]
		return row, ok
	}
	return raw.PoBGem{}, false
}

// gemQualityFlags lists the gems above the quality Gemcutter's Prisms give: up to gemQualityCorruptedMax needs a
// corrupted gem, above it no gem can be. PoB's gem quality field takes any two digits (SkillsTab.lua:864-878).
func gemQualityFlags(b *pob.PathOfBuilding) []gemFlag {
	var out []gemFlag
	eachSetGem(b, func(set string, group int, gem pob.Gem) {
		natural := gemNaturalLevel(gem)
		switch {
		case gem.Quality > gemQualityCorruptedMax:
			out = append(out, gemFlag{Set: set, Group: group, Gem: gem.NameSpec, Quality: gem.Quality, Note: fmt.Sprintf("above the game's %d%% maximum (corrupted)", gemQualityCorruptedMax)})
		case gem.Quality > gemQualityMax && natural > 0 && gem.Level > natural:
			out = append(out, gemFlag{Set: set, Group: group, Gem: gem.NameSpec, Level: gem.Level, Quality: gem.Quality,
				Note: fmt.Sprintf("level %d and quality %d%% each need a corruption; a gem is corrupted once", gem.Level, gem.Quality)})
		case gem.Quality > gemQualityMax:
			out = append(out, gemFlag{Set: set, Group: group, Gem: gem.NameSpec, Quality: gem.Quality, Note: "needs a corrupted gem"})
		}
	})
	return out
}

// gemNaturalLevel is the gem's natural maximum level (Gems.lua's naturalMaxLevel), 0 when the gem is unknown.
func gemNaturalLevel(gem pob.Gem) int {
	if row, ok := gemVariantRow(gem); ok {
		return row.NaturalMaxLevel
	}
	if gem.GemID != "" {
		for _, id := range []string{gem.GemID, strings.Replace(gem.GemID, "/SupportGem", "/SkillGemSupport", 1)} {
			if row, ok := raw.Gems[id]; ok {
				return row.NaturalMaxLevel
			}
		}
	}
	key := gem.NameSpec
	if v, ok := naturalByName.Load(key); ok {
		return v.(int)
	}
	n := 0
	if row, ok := gemRowByName(gem.NameSpec); ok {
		n = row.NaturalMaxLevel
	}
	naturalByName.Store(key, n)
	return n
}

var naturalByName sync.Map

// gemLevelFlags lists the gems above their natural maximum level: one level more needs a corrupted gem, more than one
// no gem can have. A gem whose quality also needs the corruption is gemQualityFlags' to name.
func gemLevelFlags(b *pob.PathOfBuilding) []gemFlag {
	var out []gemFlag
	eachSetGem(b, func(set string, group int, gem pob.Gem) {
		natural := gemNaturalLevel(gem)
		note := "needs a corrupted gem"
		switch {
		case natural == 0 || gem.Level <= natural:
			return
		case gem.Level > natural+1:
			note = fmt.Sprintf("above the game's maximum level %d (corrupted)", natural+1)
		case gem.Quality > gemQualityMax && gem.Quality <= gemQualityCorruptedMax:
			return
		}
		out = append(out, gemFlag{Set: set, Group: group, Gem: gem.NameSpec, Level: gem.Level, Note: note})
	})
	return out
}

// gemLevelLines folds level flags into one line per set, level and note, naming each gem with its groups.
func gemLevelLines(in []gemFlag) []string {
	var keys []string
	gems := map[string][]string{}
	for _, f := range mergeGemFlags(in) {
		k := fmt.Sprintf("level %d %s", f.Level, f.Note)
		if f.Set != "" {
			k = "set " + f.Set + ": " + k
		}
		if _, ok := gems[k]; !ok {
			keys = append(keys, k)
		}
		groups := []string{strconv.Itoa(f.Group)}
		for _, g := range f.AlsoGroups {
			groups = append(groups, strconv.Itoa(g))
		}
		gems[k] = append(gems[k], fmt.Sprintf("%s (%s)", f.Gem, strings.Join(groups, ", ")))
	}
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+": "+strings.Join(gems[k], ", "))
	}
	return out
}

// legacyGemFlags lists the gems PoB marks "Legacy Gem: only exists in Standard League" (GemTooltip.lua:242-246).
func legacyGemFlags(b *pob.PathOfBuilding) []gemFlag {
	var out []gemFlag
	eachSetGem(b, func(set string, group int, gem pob.Gem) {
		skill, ok := raw.PoBSkills[gemSkillID(gem)]
		if !ok || !skill.Legacy {
			return
		}
		note := "legacy: Standard league only"
		if base, ok := raw.PoBSkills[skill.PlusVersionOf]; ok && base.Name != "" {
			note += "; the league gem is " + base.Name
		}
		out = append(out, gemFlag{Set: set, Group: group, Gem: gem.NameSpec, Note: note})
	})
	return out
}

// delveFlag is an equipped item's mod that only Delve gives: a drop-restricted mod or a fossil one.
type delveFlag struct {
	Slot   string `json:"slot"`
	Item   string `json:"item"`
	Mod    string `json:"mod"`
	Source string `json:"source"`
}

// delveModFlags lists the explicit mods of equipped magic and rare items that match only Delve's mods for the base:
// the drop-restricted ones (ItemsTab.lua:3513, 3707-3709) and the fossil ones, PoB's Subterranean / of the Underground
// affixes (ItemsTab.lua:3710-3720, Item.lua:1767-1769), and no mod the item's other modifier sources offer.
func delveModFlags(b *pob.PathOfBuilding) []delveFlag {
	var out []delveFlag
	for _, e := range equippedItems(b) {
		item := calculator.ParseItem(e.text)
		if item.Base == nil || (item.Rarity != calculator.RarityMagic && item.Rarity != calculator.RarityRare) {
			continue
		}
		choices, ok := calculator.ItemCustomModChoices(modChoicesText(e.text))
		if !ok {
			continue
		}
		drop, fossil, other := map[string]bool{}, map[string]bool{}, map[string]bool{}
		fossilIDs := map[string]bool{}
		for _, m := range choices.Mods["FOSSIL"] {
			fossilIDs[m.ID] = true
		}
		for src, mods := range choices.Mods {
			for _, m := range mods {
				into := other
				switch {
				case src == "DELVE":
					into = drop
				case src == "FOSSIL" || fossilIDs[m.ID]:
					into = fossil
				}
				into[affixTemplate(strings.Join(m.Lines, "\n"))] = true
				if src == "DELVE" || src == "FOSSIL" || fossilIDs[m.ID] {
					continue
				}
				for _, l := range m.Lines {
					other[affixTemplate(l)] = true
				}
			}
		}
		if len(drop)+len(fossil) == 0 {
			continue
		}
		flag := func(text string) bool {
			t := affixTemplate(text)
			if other[t] || (!drop[t] && !fossil[t]) {
				return false
			}
			source := "Delve drop-restricted"
			switch {
			case drop[t] && fossil[t]:
				source = "Delve drop-restricted or fossil"
			case fossil[t]:
				source = "Delve fossil"
			}
			out = append(out, delveFlag{e.slot, item.Name, strings.ReplaceAll(text, "\n", " / "), source})
			return true
		}
		lines := item.ExplicitModLines
		for i := 0; i < len(lines); i++ {
			if lines[i].Crafted {
				continue
			}
			if i+1 < len(lines) && flag(lines[i].Line+"\n"+lines[i+1].Line) {
				i++
				continue
			}
			flag(lines[i].Line)
		}
	}
	return out
}

// gemWarnings are the warning lines for impossible gem levels and quality and for legacy gems.
func gemWarnings(b *pob.PathOfBuilding) []string {
	var out []string
	for _, f := range gemQualityFlags(b) {
		switch {
		case f.Set != "":
		case f.Quality > gemQualityCorruptedMax:
			out = append(out, fmt.Sprintf("%s in socket group %d has %d%% quality, above the game's %d%% maximum", f.Gem, f.Group, f.Quality, gemQualityCorruptedMax))
		case f.Level > 0:
			out = append(out, fmt.Sprintf("%s in socket group %d: %s", f.Gem, f.Group, f.Note))
		}
	}
	for _, f := range gemLevelFlags(b) {
		if f.Set == "" && f.Note != "needs a corrupted gem" {
			out = append(out, fmt.Sprintf("%s in socket group %d is level %d, %s", f.Gem, f.Group, f.Level, f.Note))
		}
	}
	for _, f := range legacyGemFlags(b) {
		if f.Set != "" {
			continue
		}
		out = append(out, fmt.Sprintf("%s in socket group %d is a legacy gem (Standard league only)", f.Gem, f.Group))
	}
	return out
}

// auditOf is the import audit, over every skill set and config set: gem quality and levels above the uncorrupted
// maximum, legacy gems, Delve drop-only mods, the defences and offence custom modifiers supply, ignored legacy custom
// modifier text, config values above their limits and equipped items the game could not produce. Empty lists are
// left out.
func auditOf(env *calculator.Environment) map[string]any {
	out := map[string]any{}
	if l := mergeGemFlags(gemQualityFlags(env.Build)); len(l) > 0 {
		out["gem_quality"] = l
	}
	if l := mergeGemFlags(legacyGemFlags(env.Build)); len(l) > 0 {
		out["legacy_gems"] = l
	}
	if l := delveModFlags(env.Build); len(l) > 0 {
		out["delve_mods"] = l
	}
	defences, offence := customModAudit(env)
	if len(defences) > 0 {
		out["custom_mods"] = defences
	}
	if len(offence) > 0 {
		out["custom_offence"] = offence
	}
	if l := legacyCustomModWarnings(env.Build); len(l) > 0 {
		out["custom_mods_ignored"] = l
	}
	if l := configSetWarnings(env); len(l) > 0 {
		out["config_over_limit"] = l
	}
	if l := gemLevelLines(gemLevelFlags(env.Build)); len(l) > 0 {
		out["gem_levels"] = l
	}
	if l := itemLegalityFlags(env.Build); len(l) > 0 {
		out["illegal_items"] = l
	}
	return out
}

// configSet is one config set's stored inputs; the active set's are the build's own (ConfigTab.lua:1337-1362 swaps
// them in), the others are kept in their set.
type configSet struct {
	id     int
	name   string
	active bool
	inputs []pob.Input
	blocks []pob.CustomModifierBlock
}

// configSetsOf lists every config set in PoB's order, each named by its title.
func configSetsOf(b *pob.PathOfBuilding) []configSet {
	if len(b.Config.ConfigSets) == 0 {
		return []configSet{{id: 1, name: `config set "Default" (active)`, active: true, inputs: b.Config.Inputs, blocks: b.Config.CustomModifierBlocks}}
	}
	active := b.ActiveConfigSetIndex()
	out := make([]configSet, 0, len(b.Config.ConfigSets))
	for i, cs := range b.Config.ConfigSets {
		title := "Default"
		if cs.Title != nil && *cs.Title != "" {
			title = *cs.Title
		}
		set := configSet{id: cs.ID, name: fmt.Sprintf("config set %q", title), inputs: cs.Inputs, blocks: cs.CustomModifierBlocks}
		if i == active {
			set.active, set.inputs, set.blocks = true, b.Config.Inputs, b.Config.CustomModifierBlocks
			set.name += " (active)"
		}
		out = append(out, set)
	}
	return out
}

// configSetWarnings are the config values above their limits in every config set, each line naming its set; the
// maximums are the active set's calculation's, and the totem checks read the active set only.
func configSetWarnings(env *calculator.Environment) []string {
	var out []string
	for _, cs := range configSetsOf(env.Build) {
		lines := append(chargeOverWarningsIn(cs.inputs, env), configCapWarningsIn(cs.inputs, env)...)
		if cs.active {
			lines = append(lines, totemWarnings(env)...)
		}
		for _, l := range lines {
			out = append(out, cs.name+": "+l)
		}
	}
	return out
}

// withConfigSet is b calculated with config set id active, or env itself for the active set.
func withConfigSet(env *calculator.Environment, cs configSet) *calculator.Environment {
	if cs.active {
		return env
	}
	b := configClone(env.Build)
	b.SetActiveConfigSet(cs.id)
	if !hasCustomMods(b) {
		return nil
	}
	return calcEnv(b)
}

// configClone is cloneBuild with the config sets copied too, so switching sets on the copy leaves b untouched.
func configClone(b *pob.PathOfBuilding) *pob.PathOfBuilding {
	c := cloneBuild(b)
	c.Config.ConfigSets = slices.Clone(b.Config.ConfigSets)
	c.Config.CustomModifierBlocks = slices.Clone(b.Config.CustomModifierBlocks)
	return c
}

// hasCustomMods reports whether the active config set has an enabled custom modifier block with text.
func hasCustomMods(b *pob.PathOfBuilding) bool {
	for _, block := range b.GetCustomModifierBlocks() {
		if block.IsEnabled() && strings.TrimSpace(block.Text) != "" {
			return true
		}
	}
	return false
}

// customOffence is one stat with and without a config set's custom modifiers.
type customOffence struct {
	Set     string   `json:"set"`
	Stat    string   `json:"stat"`
	With    float64  `json:"with"`
	Without float64  `json:"without"`
	Change  *float64 `json:"change_pct"`
}

// customModOffence is how much env's custom modifiers (ConfigTab.lua:1162-1180) move Full DPS and the main skill's
// DPS, from one more calculation without them; a change under 0.1% is left out.
func customModOffence(env *calculator.Environment, set string) []customOffence {
	if !hasCustomMods(env.Build) {
		return nil
	}
	b := configClone(env.Build)
	b.SetCustomModifierBlocks(nil)
	without := calcEnv(b)
	var out []customOffence
	for _, k := range []string{"FullDPS", mainDPS} {
		w, _ := statValue(env, k)
		wo, _ := statValue(without, k)
		if math.Abs(w-wo) <= 0.001*math.Max(math.Abs(w), math.Abs(wo)) {
			continue
		}
		out = append(out, customOffence{Set: set, Stat: k, With: round(w), Without: round(wo), Change: pct(wo, w)})
	}
	return out
}

// customModAudit names, for every config set with custom modifiers, the defences they supply and how much offence
// they add.
func customModAudit(env *calculator.Environment) ([]string, []customOffence) {
	var defences []string
	var offence []customOffence
	for _, cs := range configSetsOf(env.Build) {
		setEnv := withConfigSet(env, cs)
		if setEnv == nil {
			continue
		}
		if l := customModDefences(setEnv); l != "" {
			defences = append(defences, cs.name+": "+l)
		}
		offence = append(offence, customModOffence(setEnv, cs.name)...)
	}
	return defences, offence
}

// legacyCustomModWarnings names each config set's legacy customMods text that PoB drops on load because the set has
// custom modifier blocks with text: only an empty block list, or one empty block, takes the legacy text in.
//
//pob:block:src/Classes/ConfigTab.lua:924-934
func legacyCustomModWarnings(b *pob.PathOfBuilding) []string {
	var out []string
	for _, cs := range configSetsOf(b) {
		legacy := ""
		for _, in := range cs.inputs {
			if in.Name == "customMods" && in.String != nil {
				legacy = *in.String
			}
		}
		if legacy == "" || len(cs.blocks) == 0 || (len(cs.blocks) == 1 && cs.blocks[0].Text == "") {
			continue
		}
		out = append(out, fmt.Sprintf("%s: the legacy customMods text %q is ignored; Path of Building reads it only when the set has no custom modifier block text", cs.name, shortText(legacy, 80)))
	}
	return out
}

func shortText(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " / ")
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
