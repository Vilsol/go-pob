package calculator

import (
	"fmt"
	"sort"

	raw2 "github.com/Vilsol/go-pob/data/raw"
)

type SpectreRow struct {
	ID      string
	Name    string
	Skills  []string
	Tooltip []string
}

//pob:block:src/Modules/Build.lua:1524-1534
func SpectreLibrary() []SpectreRow {
	ids := make([]string, 0, len(raw2.Spectres))
	for id := range raw2.Spectres {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(a, b int) bool {
		na, nb := raw2.Spectres[ids[a]].Name, raw2.Spectres[ids[b]].Name
		if na == nb {
			return ids[a] < ids[b]
		}
		return na < nb
	})
	rows := make([]SpectreRow, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, spectreRow(id))
	}
	return rows
}

func SpectreRows(ids []string) []SpectreRow {
	rows := make([]SpectreRow, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, spectreRow(id))
	}
	return rows
}

func spectreRow(id string) SpectreRow {
	minion := raw2.Spectres[id]
	if minion == nil {
		minion = raw2.Minions[id]
	}
	if minion == nil {
		return SpectreRow{ID: id, Name: id}
	}
	row := SpectreRow{ID: id, Name: minion.Name, Tooltip: spectreTooltip(minion)}
	for _, skillID := range minion.SkillList {
		if skill, ok := raw2.PoBSkills[skillID]; ok {
			row.Skills = append(row.Skills, skill.Name)
		}
	}
	return row
}

//pob:src/Classes/MinionListControl.lua:52-81
func spectreTooltip(minion *raw2.MinionInfo) []string {
	lines := []string{
		"^7" + minion.Name,
		fmt.Sprintf("^7Life Multiplier: x%s", luaFixed(minion.Life, 2)),
	}
	if minion.EnergyShield != 0 {
		lines = append(lines, fmt.Sprintf("^7Energy Shield: %d%% of base Life", luaInt(minion.EnergyShield*100)))
	}
	if minion.Armour != 0 {
		lines = append(lines, fmt.Sprintf("^7Armour Multiplier: x%s", luaFixed(1+minion.Armour, 2)))
	}
	if minion.Evasion != 0 {
		lines = append(lines, fmt.Sprintf("^7Evasion Multiplier: x%s", luaFixed(1+minion.Evasion, 2)))
	}
	lines = append(lines,
		fmt.Sprintf("^7Resistances: %s%d^7/%s%d^7/%s%d^7/%s%d", colorFire, luaInt(minion.FireResist), colorCold, luaInt(minion.ColdResist),
			colorLightning, luaInt(minion.LightningResist), colorChaos, luaInt(minion.ChaosResist)),
		fmt.Sprintf("^7Base Damage: x%s", luaFixed(minion.Damage, 2)),
		fmt.Sprintf("^7Base Attack Speed: %s", luaFixed(1/minion.AttackTime, 2)),
	)
	for _, skillID := range minion.SkillList {
		if skill, ok := raw2.PoBSkills[skillID]; ok {
			lines = append(lines, "^7Skill: "+skill.Name)
		}
	}
	return lines
}
