package data

import (
	"compress/gzip"
	"encoding/json"
	"strings"
	"sync"
)

type ItemDBGraded struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	BaseName    string `json:"baseName"`
	Rarity      string `json:"rarity"`
	League      string `json:"league"`
	Source      string `json:"source"`
	Level       int    `json:"level"`
	StrMod      int    `json:"strMod"`
	DexMod      int    `json:"dexMod"`
	IntMod      int    `json:"intMod"`
	PrimarySlot string `json:"primarySlot"`
	Variants    int    `json:"variants"`
	Explicits   int    `json:"explicits"`
}

type ItemDBEntry struct {
	Kind    string        `json:"kind"`
	Raw     string        `json:"raw"`
	Crafted string        `json:"crafted"`
	PoB     *ItemDBGraded `json:"pob"`
}

var (
	itemDBOnce sync.Once
	itemDB     struct {
		Uniques       []ItemDBEntry                `json:"uniques"`
		Rares         []ItemDBEntry                `json:"rares"`
		FoulbornMap   map[string]map[string]string `json:"foulbornMap"`
		ExclusiveMods map[string]FoulbornMod       `json:"exclusiveMods"`
		FoulbornMods  map[string]FoulbornMod       `json:"foulbornMods"`
	}
)

func loadItemDB() {
	zr, err := gzip.NewReader(strings.NewReader(itemDBGz))
	if err == nil {
		err = json.NewDecoder(zr).Decode(&itemDB)
	}
	if err != nil {
		panic(err)
	}
}

//pob:block:src/Modules/Data.lua:1349-1371
func Uniques() []ItemDBEntry {
	itemDBOnce.Do(loadItemDB)
	return itemDB.Uniques
}

//pob:block:src/Modules/Data.lua:1257-1257
func Rares() []ItemDBEntry {
	itemDBOnce.Do(loadItemDB)
	return itemDB.Rares
}

//pob:block:src/Modules/Data.lua:1373-1375
func FoulbornMap(title string) map[string]string {
	itemDBOnce.Do(loadItemDB)
	return itemDB.FoulbornMap[title]
}

type FoulbornMod struct {
	Lines   []string `json:"lines"`
	ModTags []string `json:"modTags"`
}

func FoulbornModLines(id string, mutated bool) ([]string, bool) {
	m, ok := FoulbornModOf(id, mutated)
	return m.Lines, ok
}

func FoulbornModOf(id string, mutated bool) (FoulbornMod, bool) {
	itemDBOnce.Do(loadItemDB)
	if mutated {
		m, ok := itemDB.FoulbornMods[id]
		return m, ok
	}
	m, ok := itemDB.ExclusiveMods[id]
	return m, ok
}
