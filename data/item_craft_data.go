package data

import (
	"compress/gzip"
	"encoding/json"
	"strings"
	"sync"
)

type EnchantmentSource struct {
	Name  string `json:"name"`
	Label string `json:"label"`
}

type EnchantmentTable struct {
	Sources map[string][]string            `json:"sources"`
	Skills  map[string]map[string][]string `json:"skills"`
}

type CraftBase struct {
	Implicit         string            `json:"implicit"`
	Hidden           bool              `json:"hidden"`
	CannotBeAnointed bool              `json:"cannotBeAnointed"`
	Tags             []string          `json:"tags"`
	InfluenceTags    map[string]string `json:"influenceTags"`
	ImplicitModTypes [][]string        `json:"implicitModTypes"`
	Type             string            `json:"type"`
	SubType          string            `json:"subType"`
}

type AffixMod struct {
	ID                  string    `json:"id"`
	Type                string    `json:"type"`
	Affix               string    `json:"affix"`
	Lines               []string  `json:"lines"`
	Level               int       `json:"level"`
	Group               string    `json:"group"`
	StatOrder           []float64 `json:"statOrder"`
	WeightKey           []string  `json:"weightKey"`
	WeightVal           []int     `json:"weightVal"`
	WeightMultiplierKey []string  `json:"weightMultiplierKey"`
	WeightMultiplierVal []int     `json:"weightMultiplierVal"`
	ModTags             []string  `json:"modTags"`
}

type MasterMod struct {
	AffixMod
	Types []string `json:"types"`
}

type CrucibleMod struct {
	AffixMod
	Tier         int   `json:"tier"`
	NodeLocation []int `json:"nodeLocation"`
}

type Essence struct {
	ID   string            `json:"id"`
	Name string            `json:"name"`
	Type int               `json:"type"`
	Tier int               `json:"tier"`
	Mods map[string]string `json:"mods"`
}

var (
	itemCraftOnce sync.Once
	itemCraft     struct {
		Sources        []EnchantmentSource          `json:"sources"`
		Enchantments   map[string]*EnchantmentTable `json:"enchantments"`
		Bases          map[string]CraftBase         `json:"bases"`
		AffixTables    map[string][]AffixMod        `json:"affixTables"`
		MasterMods     []MasterMod                  `json:"masterMods"`
		Essences       []Essence                    `json:"essences"`
		VeiledMods     []AffixMod                   `json:"veiledMods"`
		NecropolisMods []AffixMod                   `json:"necropolisMods"`
		BeastCraft     []AffixMod                   `json:"beastCraft"`
		Crucible       []CrucibleMod                `json:"crucible"`
		CasterCrucible []string                     `json:"casterTagCrucibleUniques"`
		MinionCrucible []string                     `json:"minionTagCrucibleUniques"`
	}
)

func loadItemCraft() {
	zr, err := gzip.NewReader(strings.NewReader(itemCraftDataGz))
	if err == nil {
		err = json.NewDecoder(zr).Decode(&itemCraft)
	}
	if err != nil {
		panic(err)
	}
}

//pob:block:src/Modules/Data.lua:626-637
func EnchantmentSources() []EnchantmentSource {
	itemCraftOnce.Do(loadItemCraft)
	return itemCraft.Sources
}

//pob:block:src/Modules/Data.lua:665-700
func Enchantments(key string) *EnchantmentTable {
	itemCraftOnce.Do(loadItemCraft)
	return itemCraft.Enchantments[key]
}

func CraftBaseInfo(name string) CraftBase {
	b, _ := CraftBaseKnown(name)
	return b
}

func CraftBaseKnown(name string) (CraftBase, bool) {
	itemCraftOnce.Do(loadItemCraft)
	b, ok := itemCraft.Bases[name]
	return b, ok
}

func AffixTable(name string) ([]AffixMod, bool) {
	itemCraftOnce.Do(loadItemCraft)
	t, ok := itemCraft.AffixTables[name]
	return t, ok
}

func MasterMods() []MasterMod {
	itemCraftOnce.Do(loadItemCraft)
	return itemCraft.MasterMods
}

func Essences() []Essence {
	itemCraftOnce.Do(loadItemCraft)
	return itemCraft.Essences
}

func VeiledMods() []AffixMod {
	itemCraftOnce.Do(loadItemCraft)
	return itemCraft.VeiledMods
}

func NecropolisMods() []AffixMod {
	itemCraftOnce.Do(loadItemCraft)
	return itemCraft.NecropolisMods
}

func BeastCraftMods() []AffixMod {
	itemCraftOnce.Do(loadItemCraft)
	return itemCraft.BeastCraft
}

func CrucibleMods() []CrucibleMod {
	itemCraftOnce.Do(loadItemCraft)
	return itemCraft.Crucible
}

func CrucibleTagUniques() ([]string, []string) {
	itemCraftOnce.Do(loadItemCraft)
	return itemCraft.CasterCrucible, itemCraft.MinionCrucible
}
