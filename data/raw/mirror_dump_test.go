package raw

import (
	"encoding/json"
	"os"
	"sort"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestDumpMirrorTables serialises every hand-transcribed table in this package into the normalised shape scripts/dataraw_mirror.py parses out of PoB's Lua.
func TestDumpMirrorTables(t *testing.T) {
	path := os.Getenv("MIRROR_DUMP")
	if path == "" {
		t.Skip("MIRROR_DUMP not set")
	}

	out := map[string]any{}

	minions := map[string]any{}
	for id, m := range Minions {
		if _, isSpectre := Spectres[id]; isSpectre {
			continue
		}
		minions[id] = minionRow(m)
	}
	out["Minions"] = minions

	spectres := map[string]any{}
	for id, m := range Spectres {
		spectres[id] = minionRow(m)
	}
	out["Spectres"] = spectres

	out["SkillBaseFlags"] = stringListTable(SkillBaseFlags, true)
	out["SkillAddFlags"] = stringListTable(SkillAddFlags, true)
	out["SkillMinionLists"] = stringListTable(SkillMinionLists, false)
	out["SupportAddMinionLists"] = stringListTable(SupportAddMinionLists, false)
	out["SkillAddSkillTypes"] = stringListTable(SkillAddSkillTypes, false)

	skills := map[string]any{}
	for id, s := range PoBSkills {
		skills[id] = map[string]any{
			"name":     s.Name,
			"hidden":   s.Hidden,
			"fromItem": s.FromItem,
			"fromTree": s.FromTree,
			"support":  s.Support,
		}
	}
	out["PoBSkills"] = skills

	uses := map[string]any{}
	for id, slots := range SkillMinionUses {
		names := make([]string, 0, len(slots))
		for slot, on := range slots {
			if on {
				names = append(names, slot)
			}
		}
		sort.Strings(names)
		uses[id] = names
	}
	out["SkillMinionUses"] = uses

	hasSet := map[string]any{}
	for id, v := range SkillMinionHasItemSet {
		hasSet[id] = v
	}
	out["SkillMinionHasItemSet"] = hasSet

	gems := map[string]any{}
	for id, g := range Gems {
		tags := make([]string, 0, len(g.Tags))
		for tag, on := range g.Tags {
			if on {
				tags = append(tags, tag)
			}
		}
		sort.Strings(tags)
		gems[id] = map[string]any{"name": g.Name, "tags": tags}
	}
	out["Gems"] = gems

	parts := map[string]any{}
	for id, list := range SkillParts {
		rows := make([]any, 0, len(list))
		for _, p := range list {
			flags := map[string]bool{}
			for name, v := range p.Flags {
				flags[name] = v
			}
			rows = append(rows, map[string]any{"name": p.Name, "flags": flags})
		}
		parts[id] = rows
	}
	out["SkillParts"] = parts

	hidden := map[string]any{}
	for id, h := range SkillHiddenLevels {
		levels := map[string]any{}
		for n, vals := range h.Levels {
			levels[itoa(n)] = vals
		}
		types := append([]string{}, h.SkillTypes...)
		sort.Strings(types)
		hidden[id] = map[string]any{
			"skillTypes": types,
			"stats":      append([]string{}, h.Stats...),
			"levels":     levels,
		}
	}
	out["SkillHiddenLevels"] = hidden

	baseMods := map[string]any{}
	for id, mods := range SkillBaseMods {
		baseMods[id] = modNames(mods)
	}
	out["SkillBaseMods"] = baseMods

	statMap := map[string]any{}
	for stat, sm := range SkillStatMap {
		statMap[stat] = statMapRow(sm)
	}
	out["SkillStatMap"] = statMap

	statMaps := map[string]any{}
	for id, m := range SkillStatMaps {
		rows := map[string]any{}
		for stat, sm := range m {
			rows[stat] = statMapRow(sm)
		}
		statMaps[id] = rows
	}
	out["SkillStatMaps"] = statMaps

	blob, err := json.MarshalIndent(out, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, blob, 0o600); err != nil {
		t.Fatal(err)
	}
}

func stringListTable(src map[string][]string, sorted bool) map[string]any {
	out := map[string]any{}
	for id, list := range src {
		row := append([]string{}, list...)
		if sorted {
			sort.Strings(row)
		}
		out[id] = row
	}
	return out
}

func minionRow(m *MinionInfo) map[string]any {
	return map[string]any{
		"name":            m.Name,
		"life":            m.Life,
		"energyShield":    m.EnergyShield,
		"armour":          m.Armour,
		"fireResist":      m.FireResist,
		"coldResist":      m.ColdResist,
		"lightningResist": m.LightningResist,
		"chaosResist":     m.ChaosResist,
		"damage":          m.Damage,
		"damageSpread":    m.DamageSpread,
		"damageFixup":     m.DamageFixup,
		"attackTime":      m.AttackTime,
		"attackRange":     m.AttackRange,
		"accuracy":        m.Accuracy,
		"weaponType1":     m.WeaponType1,
		"weaponType2":     m.WeaponType2,
		"limit":           m.Limit,
		"skillList":       m.SkillList,
		"modList":         modNames(m.ModList),
	}
}

func statMapRow(sm *StatMap) map[string]any {
	row := map[string]any{"mods": modNames(sm.Mods)}
	for key, ptr := range map[string]*float64{
		"value": sm.Value, "mult": sm.Mult, "div": sm.Div, "base": sm.Base,
	} {
		if ptr != nil {
			row[key] = *ptr
		}
	}
	return row
}

func modNames(mods []mod.Mod) []string {
	out := make([]string, 0, len(mods))
	for _, m := range mods {
		out = append(out, m.Name())
	}
	return out
}

func itoa(n int) string {
	blob, _ := json.Marshal(n)
	return string(blob)
}
