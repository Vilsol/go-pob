package calculator

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

const pobSkillTableFile = "../testdata/pob-skills/skills.json.gz"

type pobSkill struct {
	Name           string   `json:"name"`
	Loc            string   `json:"loc"`
	Hidden         bool     `json:"hidden"`
	Support        bool     `json:"support"`
	SkillTypes     []string `json:"skillTypes"`
	Stats          []string `json:"stats"`
	LevelCount     int      `json:"levelCount"`
	MaxLevelValues int      `json:"maxLevelValues"`
	BaseFlags      []string `json:"baseFlags"`
}

type pobSkillTable struct {
	SkillCount        int                 `json:"skillCount"`
	HiddenCount       int                 `json:"hiddenCount"`
	SupportCount      int                 `json:"supportCount"`
	NoStatsCount      int                 `json:"noStatsCount"`
	NoSkillTypesCount int                 `json:"noSkillTypesCount"`
	Skills            map[string]pobSkill `json:"skills"`
}

var loadPoBSkillTable = sync.OnceValues(func() (*pobSkillTable, error) {
	return readPoBSkillTable(pobSkillTableFile)
})

func readPoBSkillTable(path string) (*pobSkillTable, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %w\nRegenerate it with `task pob-skill-table` "+
			"(runs oracle/skill_table.lua against the pinned PoB)", path, err)
	}
	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("%s is not gzip: %w", path, err)
	}
	defer func() { _ = gz.Close() }()

	var table pobSkillTable
	if err := json.NewDecoder(gz).Decode(&table); err != nil {
		return nil, fmt.Errorf("%s is malformed: %w", path, err)
	}
	if len(table.Skills) == 0 {
		return nil, fmt.Errorf("%s decoded to ZERO skills — the artifact is empty or the schema "+
			"moved. Refusing rather than comparing against nothing", path)
	}
	return &table, nil
}

func pobSkillIDFor(id string) string {
	if alias, ok := skillFlagIDAliases[id]; ok {
		return alias
	}
	return id
}
