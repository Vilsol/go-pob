package poe2skills

import (
	"encoding/json"
	"sync"

	"github.com/Vilsol/go-pob/data/poe2"
)

// statMapMu guards what reading a stat map changes: PoB's fallback copies the global entry into the set's
// map and may set hasGlobalEffect.
var statMapMu sync.Mutex

func decodeMod(s string) Mod {
	var m Mod
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		panic("poe2skills: an overlay mod is not JSON: " + err.Error())
	}
	return m
}

func modTags(m Mod) []any {
	tags, _ := m["tags"].([]any)
	return tags
}

// processMod adds a mod's source, notes a GlobalEffect tag on the effect, and tags a stat the game
// cannot grant to minions with the parent ActorCondition. A group passes through it as PoB's
// metatable passes one: its members are read as its tags.
//
//pob2:src/Modules/Data.lua:72-98
func processMod(ge *GrantedEffect, mod Mod, statName string) {
	mod["source"] = ge.ModSource
	if v, ok := mod["value"].(map[string]any); ok {
		if inner, ok := v["mod"].(map[string]any); ok {
			inner["source"] = "Skill:" + ge.ID
		}
	}
	for _, raw := range modTags(mod) {
		if tag, ok := raw.(map[string]any); ok && tag["type"] == "GlobalEffect" {
			ge.HasGlobalEffect = true
			break
		}
	}
	notMinionStat := false
	for _, set := range ge.StatSets {
		if set.NotMinionStat != nil && statName != "" && (ge.Support || ge.SkillTypes != nil && ge.SkillTypes["Buff"]) {
			for _, name := range set.NotMinionStat {
				if name == statName {
					notMinionStat = true
				}
			}
		}
	}
	if notMinionStat {
		mod["tags"] = append(modTags(mod), map[string]any{"type": "ActorCondition", "actor": "parent", "neg": true})
	}
}

// processGroup is processMod on a group: the metatable hands the group itself to processMod.
//
//pob2:block:src/Modules/Data.lua:919-921
func processGroup(ge *GrantedEffect, item *StatMapItem, statName string) {
	item.GroupSource = ge.ModSource
	notMinionStat := false
	for _, set := range ge.StatSets {
		if set.NotMinionStat != nil && statName != "" && (ge.Support || ge.SkillTypes != nil && ge.SkillTypes["Buff"]) {
			for _, name := range set.NotMinionStat {
				if name == statName {
					notMinionStat = true
				}
			}
		}
	}
	if notMinionStat {
		item.Group = append(item.Group, Mod{"type": "ActorCondition", "actor": "parent", "neg": true})
	}
}

func newStatMap(entry *poe2.StatMapEntry) *StatMap {
	m := &StatMap{
		Stat: entry.Stat, SkillFlag: entry.SkillFlag, Items: make([]StatMapItem, len(entry.Items)),
		Value: entry.Value, Mult: entry.Mult, Div: entry.Div, Base: entry.Base,
		HasValue: entry.HasValue, HasMult: entry.HasMult, HasDiv: entry.HasDiv, HasBase: entry.HasBase,
	}
	for i, it := range entry.Items {
		item := StatMapItem{
			Value: it.Value, Mult: it.Mult, Div: it.Div, Base: it.Base,
			HasValue: it.HasValue, HasMult: it.HasMult, HasDiv: it.HasDiv, HasBase: it.HasBase,
		}
		if it.Group == nil {
			item.Mod = decodeMod(it.Mod)
		} else {
			item.Group = make([]Mod, len(it.Group))
			for j, g := range it.Group {
				item.Group[j] = decodeMod(g)
			}
		}
		m.Items[i] = item
	}
	return m
}

// loadGrantedEffect is Data.lua's pass over one loaded skill.
//
//pob2:block:src/Modules/Data.lua:929-966
func loadGrantedEffect(ge *GrantedEffect, id string) {
	ge.Name = sanitiseText(ge.Name)
	ge.ID = id
	ge.ModSource = "Skill:" + id
	for _, set := range ge.StatSets {
		o := set.Overlay
		if o.HasBaseFlags {
			set.BaseFlags = make(map[string]bool, len(o.BaseFlags))
			for _, f := range o.BaseFlags {
				set.BaseFlags[f] = true
			}
		}
		if o.BaseMods != nil {
			set.BaseMods = make([]Mod, len(o.BaseMods))
			for i, s := range o.BaseMods {
				set.BaseMods[i] = decodeMod(s)
			}
		}
	}
	for _, set := range ge.StatSets {
		for _, m := range set.BaseMods {
			processMod(ge, m, "")
		}
	}
	for _, set := range ge.StatSets {
		set.statMap = make(map[string]*StatMap, len(set.Overlay.StatMap))
		for i := range set.Overlay.StatMap {
			entry := &set.Overlay.StatMap[i]
			m := newStatMap(entry)
			for j := range m.Items {
				if m.Items[j].Mod != nil {
					processMod(ge, m.Items[j].Mod, entry.Stat)
				} else {
					for _, g := range m.Items[j].Group {
						processMod(ge, g, entry.Stat)
					}
				}
			}
			set.statMap[entry.Stat] = m
		}
	}
}

// StatMap is statSet.statMap[stat]: the set's own entry, else a processed copy of data.skillStatMap's,
// which it keeps, as the metatable does.
//
//pob2:src/Modules/Data.lua:913-925
func (set *StatSet) StatMap(stat string) (*StatMap, bool) {
	statMapMu.Lock()
	defer statMapMu.Unlock()
	if m, ok := set.statMap[stat]; ok {
		return m, true
	}
	entry, ok := poe2.SkillStatMapEntry(stat)
	if !ok {
		return nil, false
	}
	m := newStatMap(entry)
	set.statMap[stat] = m
	for j := range m.Items {
		if m.Items[j].Mod != nil {
			processMod(set.effect, m.Items[j].Mod, stat)
		} else {
			processGroup(set.effect, &m.Items[j], stat)
		}
	}
	return m, true
}

// GlobalEffect is hasGlobalEffect: set at load by a GlobalEffect tag, and later by a stat map read that
// copies one in.
func (ge *GrantedEffect) GlobalEffect() bool {
	statMapMu.Lock()
	defer statMapMu.Unlock()
	return ge.HasGlobalEffect
}
