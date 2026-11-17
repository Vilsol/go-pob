package mcp

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

// chargeOverrides are the charge counts ConfigOptions.lua:903-935 apply as an OVERRIDE that CalcPerform.lua:1046-1092
// does not clamp to the maximum.
var chargeOverrides = []struct{ use, override, max, name string }{
	{"usePowerCharges", "overridePowerCharges", "PowerChargesMax", "Power Charges"},
	{"useFrenzyCharges", "overrideFrenzyCharges", "FrenzyChargesMax", "Frenzy Charges"},
	{"useEnduranceCharges", "overrideEnduranceCharges", "EnduranceChargesMax", "Endurance Charges"},
	{"useSiphoningCharges", "overrideSiphoningCharges", "SiphoningChargesMax", "Siphoning Charges"},
	{"useChallengerCharges", "overrideChallengerCharges", "ChallengerChargesMax", "Challenger Charges"},
	{"useBlitzCharges", "overrideBlitzCharges", "BlitzChargesMax", "Blitz Charges"},
}

// validationWarnings are the build checks PoB's sidebar does not make: links per socket group, passive points
// over the character level (Build.lua:898-939's Required Level) and charge overrides above the maximum.
func validationWarnings(env *calculator.Environment) []string {
	b := env.Build
	var out []string
	for _, sm := range socketMap(b) {
		total := 0
		for _, g := range sm.groups {
			total += g.gems
			if g.gems > sm.links {
				out = append(out, fmt.Sprintf("Too few links: the largest link group in %s is %d, socket group %d (%s) has %d gems",
					sm.slot, sm.links, g.index, g.name, g.gems))
			}
		}
		// Build.lua:2067-2069's warning, also for gems the engine's game data lacks and CalcSetup.lua:1356 counts
		if total > sm.gemSockets {
			out = append(out, "You have too many gems in your "+sm.slot+" slot")
		}
	}
	if p := pointsOf(env); p.LevelNeeded > b.Build.Level {
		out = append(out, fmt.Sprintf("Your allocated passive points need character level %d (level is %d)", p.LevelNeeded, b.Build.Level))
	}
	return append(out, chargeOverWarnings(env)...)
}

// chargeOverWarnings names the active config set's charge overrides above the maximum.
func chargeOverWarnings(env *calculator.Environment) []string {
	return chargeOverWarningsIn(env.Build.Config.Inputs, env)
}

// chargeOverWarningsIn names the charge overrides in inputs above env's maximum.
func chargeOverWarningsIn(list []pob.Input, env *calculator.Environment) []string {
	var out []string
	inputs := map[string]pob.Input{}
	for _, in := range list {
		inputs[in.Name] = in
	}
	for _, c := range chargeOverrides {
		use, over := inputs[c.use], inputs[c.override]
		if use.Boolean == nil || !*use.Boolean || over.Number == nil {
			continue
		}
		if m, ok := env.Player.Output[c.max]; ok && *over.Number > m {
			out = append(out, fmt.Sprintf("Config %s is %g, above your maximum of %g %s; Path of Building does not cap it",
				c.override, *over.Number, m, c.name))
		}
	}
	return out
}

// configCaps are the count options ConfigOptions.lua clamps to a fixed maximum when applying them (305-306, 321-324,
// 376-379, 382-383, 437-438, 517-518, 663-664, 719-720).
var configCaps = []struct {
	name string
	max  float64
}{
	{"barkskinStacks", 10}, {"bloodsoakedBannerStages", 10}, {"conditionCorruptingCryStages", 10}, {"overrideCruelty", 40},
	{"touchedDebuffsCount", 10}, {"OverloadedIntensity", 3}, {"shardsConsumed", 12}, {"configResonanceCount", 50},
}

// configCapWarningsIn names the stored config values in inputs above the maximum PoB clamps them to: the fixed caps
// above and Rage, which CalcPerform.lua:914-922 caps at MaximumRage and ignores when nothing in the build lets it gain
// Rage (no Condition:CanGainRage flag and no Rage regeneration; the option's own ifFlag, ConfigOptions.lua:1073-1074).
func configCapWarningsIn(inputs []pob.Input, env *calculator.Environment) []string {
	var out []string
	warn := func(name string, v, m float64, what string) {
		out = append(out, fmt.Sprintf("Config %s is %g, above the maximum of %g%s; Path of Building uses %g", name, v, m, what, m))
	}
	for _, in := range inputs {
		if in.Number == nil {
			continue
		}
		if in.Name == "multiplierRage" {
			m, ok := env.Player.Output["MaximumRage"]
			switch {
			case !ok && *in.Number > 0:
				out = append(out, fmt.Sprintf("Config %s is %g, but no Rage source found in the build (nothing grants Condition:CanGainRage or Rage regeneration); Path of Building ignores it", in.Name, *in.Number))
			case ok && *in.Number > m:
				warn(in.Name, *in.Number, m, " Rage")
			}
			continue
		}
		for _, c := range configCaps {
			if c.name == in.Name && *in.Number > c.max {
				warn(in.Name, *in.Number, c.max, "")
			}
		}
	}
	return out
}

// itemNonModLines parse as mod lines with extra text but are not stats: Item.lua:546-555's flag lines, which become
// mod lines when a variant prefix stops the exact match, and ModMaster.lua:732-734's crafting-bench lines.
var itemNonModLines = map[string]bool{
	"Split": true, "Mirrored": true, "Corrupted": true, "Fractured Item": true, "Synthesised Item": true,
	"Prefixes Cannot Be Changed": true, "Suffixes Cannot Be Changed": true, "Can have up to 3 Crafted Modifiers": true,
}

// customDefences are the defences customModDefences totals, with the mod names CalcSections.lua tabulates for each
// (1510, 1556-1559, 1582, 1642, 1658-1670, 1677, 1685, 1743, 1747) and the block maximums.
var customDefences = []struct {
	label string
	names []string
	types []string
	pct   bool
}{
	{"Fire Res", []string{"FireResist", "ElementalResist"}, []string{"BASE"}, true},
	{"Cold Res", []string{"ColdResist", "ElementalResist"}, []string{"BASE"}, true},
	{"Lightning Res", []string{"LightningResist", "ElementalResist"}, []string{"BASE"}, true},
	{"Chaos Res", []string{"ChaosResist"}, []string{"BASE"}, true},
	{"Fire Res Max", []string{"FireResistMax", "ElementalResistMax"}, []string{"BASE"}, true},
	{"Cold Res Max", []string{"ColdResistMax", "ElementalResistMax"}, []string{"BASE"}, true},
	{"Lightning Res Max", []string{"LightningResistMax", "ElementalResistMax"}, []string{"BASE"}, true},
	{"Chaos Res Max", []string{"ChaosResistMax"}, []string{"BASE"}, true},
	{"Phys Damage Reduction", []string{"PhysicalDamageReduction", "PhysicalDamageReductionWhenHit"}, []string{"BASE"}, true},
	{"Life", []string{"Life"}, []string{"BASE", "INC"}, false},
	{"Life Regen", []string{"LifeRegen"}, []string{"BASE"}, false},
	{"Life Regen", []string{"LifeRegenPercent"}, []string{"BASE"}, true},
	{"Mana", []string{"Mana"}, []string{"BASE", "INC"}, false},
	{"Energy Shield", []string{"EnergyShield", "Defences"}, []string{"BASE", "INC"}, false},
	{"Ward", []string{"Ward"}, []string{"BASE", "INC"}, false},
	{"Armour", []string{"Armour", "ArmourAndEvasion", "Defences"}, []string{"BASE", "INC"}, false},
	{"Evasion", []string{"Evasion", "ArmourAndEvasion", "Defences"}, []string{"BASE", "INC"}, false},
	{"Block", []string{"BlockChance"}, []string{"BASE"}, true},
	{"Spell Block", []string{"SpellBlockChance"}, []string{"BASE"}, true},
	{"Block Max", []string{"BlockChanceMax"}, []string{"BASE"}, true},
	{"Spell Block Max", []string{"SpellBlockChanceMax"}, []string{"BASE"}, true},
	{"Spell Suppression", []string{"SpellSuppressionChance"}, []string{"BASE"}, true},
}

// customModDefences names the resistances and defences that custom modifiers (ConfigTab.lua:1162-1180, source
// "Custom:<block>") supply, so a build that leans on them is not mistaken for one its gear and tree support.
func customModDefences(env *calculator.Environment) string {
	if len(env.Build.GetCustomModifierBlocks()) == 0 {
		return ""
	}
	var parts []string
	for _, d := range customDefences {
		for _, t := range d.types {
			var sum float64
			for _, r := range env.CalcsModSection(calculator.ModSectionData{ModName: d.names, ModType: t}, false) {
				if !strings.HasPrefix(r.Source, "Custom") {
					continue
				}
				var v float64
				_, _ = fmt.Sscan(strings.TrimPrefix(r.Value, "+"), &v)
				sum += v
			}
			if sum == 0 {
				continue
			}
			unit := ""
			switch {
			case t == "INC":
				unit = "% inc"
			case d.pct:
				unit = "%"
			}
			parts = append(parts, fmt.Sprintf("%s %+g%s", d.label, round(sum), unit))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "Custom modifiers supply " + strings.Join(parts, ", ") + "; the gear and tree do not"
}

var nodeParseErrorRe = regexp.MustCompile(`^Error parsing (.+?) mod: .*, stat text: (.+), with \d+ mods found$`)

// unparsedWarnings lists every mod line the calculation ignores because it does not parse, with its source: tree
// nodes (unparsedNodeLines; the calculation caches node mod lists without their errors), mastery effects, cluster and
// conquered nodes (never cached, so the environment's DebugErrors carries them), equipped items and tree
// jewels (Item.lua:1325-1352's modLine.extra, drawn red by ItemsTab.lua), and custom modifiers (ConfigTab.lua:1169-1172).
func unparsedWarnings(env *calculator.Environment) []string {
	var out []string
	add := func(source, line string) {
		out = append(out, fmt.Sprintf("Unsupported mod in %s: %s", source, line))
	}
	b := env.Build
	v := specTreeVersion(b)
	for _, id := range b.Build.PassiveNodes {
		if n, ok := treeNodeByID(v, id); ok {
			for _, line := range unparsedNodeLines(n.Stats) {
				add("Passive Node ("+n.Name+")", line)
			}
		}
	}
	for _, d := range env.DebugErrors {
		if m := nodeParseErrorRe.FindStringSubmatch(d); m != nil {
			add(m[1], m[2])
		}
	}
	itemLines := func(source, text string) {
		item := calculator.ParseItem(text)
		for _, list := range [][]calculator.ModLine{item.EnchantModLines, item.ScourgeModLines, item.ImplicitModLines, item.ExplicitModLines, item.CrucibleModLines} {
			for _, ml := range list {
				if ml.Extra != "" && item.ItemModLineVisible(ml) && !itemNonModLines[ml.Line] {
					add(fmt.Sprintf("%s (%s)", source, item.Name), ml.Line)
				}
			}
		}
	}
	for _, e := range equippedItems(b) {
		itemLines(e.slot, e.text)
	}
	if spec := activeSpec(b); spec != nil {
		texts := map[int]string{}
		for _, it := range b.Items.Items {
			texts[it.ID] = it.Text
		}
		for _, sk := range spec.Sockets {
			if text, ok := texts[sk.ItemID]; ok && sk.ItemID != 0 {
				itemLines("jewel socket "+strconv.Itoa(sk.NodeID), text)
			}
		}
	}
	for _, block := range b.GetCustomModifierBlocks() {
		if !block.IsEnabled() {
			continue
		}
		for _, line := range strings.Split(block.Text, "\n") {
			// a "#" line parses to nothing, so ConfigTab.lua:1171-1172 skips it like any unparsed line; it is a note
			if t := strings.TrimSpace(line); t == "" || strings.HasPrefix(t, "#") {
				continue
			}
			if entry := calculator.ParseCustomModLine(line); entry == nil || entry.ModList == nil || entry.Extra != "" {
				add("custom modifiers", strings.TrimSpace(line))
			}
		}
	}
	return out
}

// unparsedNodeLines are a node's stat lines PassiveTree:ProcessStats leaves with no mod list or with extra text,
// after trying to combine each with the lines that follow it.
//
//pob:block:src/Classes/PassiveTree.lua:854-898
func unparsedNodeLines(stats []string) []string {
	var sd []string
	for _, l := range stats {
		for _, part := range strings.Split(l, "\n") {
			if part != "" {
				sd = append(sd, part)
			}
		}
	}
	var out []string
	for i := 0; i < len(sd); i++ {
		entry := calculator.ParseMod(sd[i], false)
		if entry.ModList == nil || entry.Extra != "" {
			for endI := i + 1; endI < len(sd); endI++ {
				comb := strings.Join(sd[i:endI+1], " ")
				if c := calculator.ParseMod(comb, true); c.ModList != nil && c.Extra == "" {
					entry = c
					i = endI
					break
				}
			}
		}
		if entry.ModList == nil || entry.Extra != "" {
			out = append(out, sd[i])
		}
	}
	return out
}

type socketGroupUse struct {
	index int
	name  string
	gems  int
}

type itemSockets struct {
	slot       string
	sockets    string
	gemSockets int
	links      int
	groups     []socketGroupUse
}

// socketMap lists each equipped item with sockets, its gem sockets (CalcSetup.lua:1322-1330 leaves abyssal ones out)
// and the enabled socket groups placed in it, counting enabled gems as CalcSetup.lua:1311-1321 does.
func socketMap(b *pob.PathOfBuilding) []itemSockets {
	groups := activeSkills(b)
	var out []itemSockets
	for _, e := range equippedItems(b) {
		item := calculator.ParseItem(e.text)
		if len(item.Sockets) == 0 {
			continue
		}
		row := itemSockets{slot: e.slot, sockets: socketString(item), links: maxLinks(item)}
		for _, sk := range item.Sockets {
			switch sk.Color {
			case "R", "G", "B", "W":
				row.gemSockets++
			}
		}
		for i, g := range groups {
			if !g.Enabled || g.Slot != e.slot || g.Source != "" {
				continue
			}
			use := socketGroupUse{index: i + 1}
			for _, gem := range g.Gems {
				if !gem.Enabled {
					continue
				}
				if use.name == "" {
					use.name = gem.NameSpec
				}
				use.gems++
			}
			if use.gems > 0 {
				row.groups = append(row.groups, use)
			}
		}
		out = append(out, row)
	}
	return out
}

func socketsSection(b *pob.PathOfBuilding) []any {
	list := socketMap(b)
	out := make([]any, 0, len(list))
	for _, sm := range list {
		gems := 0
		var groups []any
		for _, g := range sm.groups {
			gems += g.gems
			groups = append(groups, map[string]any{"group": g.index, "name": g.name, "gems": g.gems})
		}
		out = append(out, map[string]any{"slot": sm.slot, "sockets": sm.sockets, "links": sm.links, "gems": gems, "groups": nonNil(groups)})
	}
	return out
}
