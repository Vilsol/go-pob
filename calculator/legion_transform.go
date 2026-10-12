package calculator

import (
	"strconv"
	"strings"
	"sync"

	tjcalc "github.com/Vilsol/timeless-jewels/calculator"
	tjdata "github.com/Vilsol/timeless-jewels/data"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
)

func legionConqueror(jewelType tjdata.JewelType, c mod.ConquerorType) (tjdata.Conqueror, bool) {
	idStr, version := c.ID, uint32(0)
	if suffix := strings.Index(idStr, "_v"); suffix >= 0 {
		v, err := strconv.Atoi(idStr[suffix+2:])
		if err != nil || v < 1 {
			return "", false
		}
		version = uint32(v - 1) //nolint:gosec // guarded above: v >= 1
		idStr = idStr[:suffix]
	}

	index, err := strconv.Atoi(idStr)
	if err != nil || index < 0 {
		return "", false
	}

	for name, entry := range tjdata.TimelessJewelConquerors[jewelType] {
		if entry.Index == uint32(index) && entry.Version == version { //nolint:gosec // guarded above
			return name, true
		}
	}
	return "", false
}

//pob:block:src/Classes/PassiveSpec.lua:1099-1111
var timelessJewelTypeByConqueror = map[string]int{
	"vaal": 1, "karui": 2, "maraketh": 3, "templar": 4, "eternal": 5, "kalguur": 6,
	"abyss_murderous": 7, "abyss_searching": 8, "abyss_hypnotic": 9,
	"abyss_ghastly": 10, "abyss_special": 11,
}

func legionJewelType(conquerorType string) int {
	return timelessJewelTypeByConqueror[conquerorType]
}

func legionJewelTypeFor(conquerorType string) (tjdata.JewelType, bool) {
	key := legionJewelType(conquerorType)
	if key <= 0 || key >= 7 {
		return 0, false
	}
	return tjdata.JewelType(key), true //nolint:gosec // bounded 1..6 above
}

//pob:block:src/Export/Scripts/legionPassives.lua:24-56
var legionDatErrorRenames = map[string]string{
	"karui_notable_add_physical_taken_as_fire": "karui_notable_add_rage_on_melee_hit",
	"karui_notable_add_faster_burn":            "karui_notable_add_faster_ignite",
	"maraketh_notable_add_ailment_avoid":       "maraketh_notable_add_stun_avoid",
	"maraketh_notable_add_flask_effect":        "maraketh_notable_add_alchemists_genius",
}

//pob:block:src/Export/Scripts/legionPassives.lua:8-23
var legionDatErrorNameSwaps = map[string]struct{ name, id string }{
	"templar_notable_minimum_frenzy_charge": {"Powerful Faith", "templar_notable_minimum_power_charge"},
	"templar_notable_minimum_power_charge":  {"Frenzied Faith", "templar_notable_minimum_frenzy_charge"},
}

func legionNodeRowID(s *tjdata.AlternatePassiveSkill) string {
	if swap, ok := legionDatErrorNameSwaps[s.ID]; ok && swap.name == s.Name {
		return swap.id
	}
	return legionRowID(s.ID)
}

func legionRowID(id string) string {
	if renamed, ok := legionDatErrorRenames[id]; ok {
		return renamed
	}
	return id
}

type legionReplacementKey struct {
	jewel   mod.LegionJewel
	graphID string
}

type legionReplacementResult struct {
	replaceWith *data.LegionNode
	added       []string
}

var legionReplacements sync.Map

func legionReplacement(jewel mod.LegionJewel, graphID string) (*data.LegionNode, []string) {
	key := legionReplacementKey{jewel: jewel, graphID: graphID}
	if hit, ok := legionReplacements.Load(key); ok {
		r := hit.(legionReplacementResult)
		return r.replaceWith, r.added
	}
	replaceWith, added := computeLegionReplacement(jewel, graphID)
	if err := raw.EnsureTimelessSources(); err == nil {
		legionReplacements.Store(key, legionReplacementResult{replaceWith: replaceWith, added: added})
	}
	return replaceWith, added
}

func computeLegionReplacement(jewel mod.LegionJewel, graphID string) (*data.LegionNode, []string) {
	if err := raw.EnsureTimelessSources(); err != nil {
		return nil, nil
	}

	jewelType, ok := legionJewelTypeFor(jewel.Conqueror.Type)
	if !ok {
		return nil, nil
	}
	conqueror, ok := legionConqueror(jewelType, jewel.Conqueror)
	if !ok {
		return nil, nil
	}

	id, err := strconv.ParseInt(graphID, 10, 64)
	if err != nil {
		return nil, nil
	}
	skill := raw.PassiveSkillByGraphID(id)
	if skill == nil || skill.Key < 0 {
		return nil, nil
	}

	if jewel.ID < 0 {
		return nil, nil
	}
	result := tjcalc.Calculate(uint32(skill.Key), uint32(jewel.ID), jewelType, conqueror) //nolint:gosec // both guarded above

	return renderTimelessResult(result, substituteRolls)
}

func renderTimelessResult(result tjdata.AlternatePassiveSkillInformation, render func(data.LegionNode, tjdata.StatRolls) []string) (*data.LegionNode, []string) {
	var replaceWith *data.LegionNode
	if result.AlternatePassiveSkill != nil {
		if node, found := data.LegionNodes[legionNodeRowID(result.AlternatePassiveSkill)]; found {
			resolved := node
			resolved.Stats = render(node, result.StatRolls)
			replaceWith = &resolved
		}
	}

	var added []string
	for _, addition := range result.AlternatePassiveAdditionInformations {
		if addition.AlternatePassiveAddition == nil {
			continue
		}
		node, found := data.LegionAdditions[legionRowID(addition.AlternatePassiveAddition.ID)]
		if !found {
			continue
		}
		added = append(added, render(node, addition.StatRolls)...)
	}

	return replaceWith, added
}

//pob:block:src/Classes/PassiveSpec.lua:1257-1261
func substituteRolls(node data.LegionNode, rolls tjdata.StatRolls) []string {
	if len(node.Stats) == 0 {
		return nil
	}

	out := make([]string, 0, len(node.Stats))
	for i, template := range node.Stats {
		if i >= len(node.Rolls) {
			out = append(out, template)
			continue
		}
		stat := node.Rolls[i]
		if stat.Index < 1 || stat.Index > len(rolls) {
			out = append(out, template)
			continue
		}
		out = append(out, legionStatLine(template, stat, float64(rolls[stat.Index-1])))
	}
	return out
}

//pob:block:src/Classes/PassiveSpec.lua:1229-1233
func substituteAbyssRolls(node data.LegionNode, rolls tjdata.StatRolls, first string) []string {
	if len(node.Stats) == 0 {
		return nil
	}
	order := node.Rolls
	if first != "" {
		order = pinPairsOrder(node.Rolls, []string{first}, func(s data.LegionStat) string { return s.Key })
	}
	out := make([]string, 0, len(node.Stats))
	for _, statLine := range node.Stats {
		for _, stat := range order {
			if stat.Index < 1 || stat.Index > len(rolls) {
				continue
			}
			statLine = legionStatLine(statLine, stat, float64(rolls[stat.Index-1]))
		}
		out = append(out, statLine)
	}
	return out
}

const pairsPickAbyssStats = "abyssStats"

type abyssStatTie struct {
	legionID string
	rolls    map[string]float64
}

func abyssStatTieOf(node data.LegionNode, rolls tjdata.StatRolls) (abyssStatTie, bool) {
	tie := abyssStatTie{legionID: node.ID, rolls: map[string]float64{}}
	for i, a := range node.Rolls {
		for j, b := range node.Rolls {
			if i == j || a.Min == a.Max || a.Min != b.Min || a.Max != b.Max ||
				a.Index < 1 || a.Index > len(rolls) || b.Index < 1 || b.Index > len(rolls) ||
				rolls[a.Index-1] == rolls[b.Index-1] {
				continue
			}
			tie.rolls[a.Key] = float64(rolls[a.Index-1])
		}
	}
	return tie, len(tie.rolls) > 0
}
