package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

var rdGeneratorsPoE2Items = []rdGenerator{
	{
		name:     "poe2-runeforging",
		source:   rdGameDataOverlay,
		out:      "data/poe2/runeforging_generated.go",
		lua:      "scripts/gopob/regendata_poe2.lua",
		dump:     "poe2-misc-data",
		luaArgs:  []string{"runeforging"},
		emit:     "poe2-data-emit",
		emitArgs: []string{"runeforging"},
		needsPob: true,
		note:     "data.runeforgingCrafts (src/Data/VerisiumCrafts.lua): the bases runeforging or runemastering turns each base into. Game data FROM go-pob-data: `gamedatadump poe2-misc-data` writes src/Data/VerisiumCrafts.lua (PoB-PoE2's Export/Scripts/miscdata.lua over 0.5's Expedition2VerisiumCrafts and BaseItemTypes), byte-identical to the pin's, and regendata_poe2.lua loads it in place of the pin's (GAMEDATA_DUMP)",
	},
	{
		name:     "poe2-item-db",
		source:   rdPoBAuthored,
		out:      "data/poe2/item_db_generated.go",
		lua:      "scripts/gopob/regendata_poe2.lua",
		luaArgs:  []string{"item-db"},
		emit:     "poe2-data-emit",
		emitArgs: []string{"item-db"},
		needsPob: true,
		note:     "the Items page's unique and rare databases as Main.lua reads them (src/Modules/Main.lua:155-192): data.uniques by key (src/Data/Uniques, with Special/Generated's uniques as Data.lua builds them) and data.rares (src/Data/Rares.lua), raw. PoB-authored item texts; Generated's are built from the pin's data.itemMods.Exclusive and data.keystones",
	},
	{
		name:     "poe2-item-mods",
		source:   rdGameDataOverlay,
		out:      "data/poe2/item_mods_generated.go",
		lua:      "scripts/gopob/regendata_poe2.lua",
		dump:     "poe2-mods",
		luaArgs:  []string{"item-mods"},
		emit:     "poe2-data-emit",
		emitArgs: []string{"item-mods"},
		needsPob: true,
		note:     "data.itemMods' affix tables (Item, Flask, Charm, Jewel, IncursionLimb, Corruption, Desecrated) after Data.lua's load (src/Modules/Data.lua:692-717); tradeHashes dropped. Game data FROM go-pob-data: `gamedatadump poe2-mods` runs PoB-PoE2's Export/Scripts/mods.lua over 0.5's Mods, Stats, ModType, ModFamily and Tags tables and stat descriptions, every row equal to the pin's src/Data/Mod*.lua up to the order of each tradeHashes table (`mise run poe2-gamedata-check`), and regendata_poe2.lua loads them in place of the pin's (GAMEDATA_DUMP)",
	},
	{
		name:     "poe2-mod-scalability",
		source:   rdGameDataOverlay,
		out:      "data/poe2/mod_scalability_generated.go",
		lua:      "scripts/gopob/regendata_poe2.lua",
		dump:     "poe2-mod-scalability",
		luaArgs:  []string{"mod-scalability"},
		emit:     "poe2-data-emit",
		emitArgs: []string{"mod-scalability"},
		needsPob: true,
		note:     "data.modScalability (src/Data/ModScalability.lua, src/Modules/Data.lua:481), the table itemLib.applyRange reads first. Game data FROM go-pob-data: `gamedatadump poe2-mod-scalability` runs PoB-PoE2's Export/Scripts/modScalability.lua over 0.5's Stats table and stat descriptions, every row equal to the pin's up to one later wording go-pob-data carries (`mise run poe2-gamedata-check`), and regendata_poe2.lua loads it in place of the pin's (GAMEDATA_DUMP)",
	},
	{
		name:     "poe2-item-mods-exclusive",
		source:   rdGameDataOverlay,
		out:      "data/poe2/item_mods_exclusive_generated.go",
		lua:      "scripts/gopob/regendata_poe2.lua",
		dump:     "poe2-mods",
		luaArgs:  []string{"item-mods-exclusive"},
		emit:     "poe2-data-emit",
		emitArgs: []string{"item-mods-exclusive"},
		needsPob: true,
		note:     "data.itemMods.Exclusive's lines and stat orders (src/Modules/Data.lua:700), which Item.lua orders an advanced-copy unique's lines by. Game data FROM go-pob-data: `gamedatadump poe2-mods` writes src/Data/ModItemExclusive.lua, every row equal to the pin's up to the order of each tradeHashes table (`mise run poe2-gamedata-check`), and regendata_poe2.lua loads it in place of the pin's (GAMEDATA_DUMP)",
	},
}

type p2eRuneforgingJSON struct {
	Base   string `json:"base"`
	Crafts []struct {
		BaseTypeID string `json:"baseTypeId"`
		Name       string `json:"name"`
	} `json:"crafts"`
}

func p2eRuneforging(body []byte) (string, error) {
	var rows []p2eRuneforgingJSON
	if err := p2eDecode(body, &rows); err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("holds no runeforging crafts")
	}
	if !sort.SliceIsSorted(rows, func(i, j int) bool { return rows[i].Base < rows[j].Base }) {
		return "", fmt.Errorf("the runeforging crafts are not sorted by base; RuneforgingCraftsOf binary-searches them")
	}
	var b strings.Builder
	p2eHeader(&b, "runeforging", "src/Modules/Data.lua:1107-1107")
	fmt.Fprintf(&b, "// RuneforgingCrafts is data.runeforgingCrafts, %d bases sorted by name, each with its crafts in order.\nvar RuneforgingCrafts = []RuneforgingBase{\n", len(rows))
	for _, r := range rows {
		if len(r.Crafts) == 0 {
			return "", fmt.Errorf("%s has no crafts", r.Base)
		}
		var crafts []string
		for _, c := range r.Crafts {
			crafts = append(crafts, "{BaseTypeID: "+strconv.Quote(c.BaseTypeID)+", Name: "+strconv.Quote(c.Name)+"}")
		}
		b.WriteString("\t{Base: " + strconv.Quote(r.Base) + ", Crafts: []RuneforgingCraft{" + strings.Join(crafts, ", ") + "}},\n")
	}
	b.WriteString("}\n")
	return b.String(), nil
}

type p2eItemDBJSON struct {
	Uniques []struct {
		Kind string   `json:"kind"`
		Raws []string `json:"raws"`
	} `json:"uniques"`
	Rares []string `json:"rares"`
}

func p2eItemDB(body []byte) (string, error) {
	var dump p2eItemDBJSON
	if err := p2eDecode(body, &dump); err != nil {
		return "", err
	}
	count := 0
	for _, u := range dump.Uniques {
		count += len(u.Raws)
	}
	if count < 100 || len(dump.Rares) < 50 {
		return "", fmt.Errorf("holds %d uniques and %d rare templates; Data.lua did not load", count, len(dump.Rares))
	}
	var b strings.Builder
	p2eHeader(&b, "item-db", "src/Modules/Data.lua:1148-1158")
	fmt.Fprintf(&b, "// UniqueDB is data.uniques, %d raw uniques by key, keys sorted.\nvar UniqueDB = []UniqueDBKind{\n", count)
	for _, u := range dump.Uniques {
		b.WriteString("\t{Kind: " + strconv.Quote(u.Kind) + ", Raws: []string{\n")
		for _, raw := range u.Raws {
			b.WriteString("\t\t" + strconv.Quote(raw) + ",\n")
		}
		b.WriteString("\t}},\n")
	}
	b.WriteString("}\n\n")
	fmt.Fprintf(&b, "// RareDB is data.rares, %d raw rare templates.\nvar RareDB = []string{\n", len(dump.Rares))
	for _, raw := range dump.Rares {
		b.WriteString("\t" + strconv.Quote(raw) + ",\n")
	}
	b.WriteString("}\n")
	return b.String(), nil
}

type p2eItemModJSON struct {
	Category            string    `json:"category"`
	ID                  string    `json:"id"`
	Type                string    `json:"type"`
	Affix               string    `json:"affix"`
	Lines               []string  `json:"lines"`
	StatOrder           []float64 `json:"statOrder"`
	Level               float64   `json:"level"`
	Group               string    `json:"group"`
	WeightKey           []string  `json:"weightKey"`
	WeightVal           []float64 `json:"weightVal"`
	WeightMultiplierKey []string  `json:"weightMultiplierKey"`
	WeightMultiplierVal []float64 `json:"weightMultiplierVal"`
	Tags                []string  `json:"tags"`
	ModTags             []string  `json:"modTags"`
	Unscalable          bool      `json:"unscalable"`
	NodeType            *float64  `json:"nodeType"`
}

func p2eItemMods(body []byte) (string, error) {
	var rows []p2eItemModJSON
	if err := p2eDecode(body, &rows); err != nil {
		return "", err
	}
	if len(rows) < 1000 {
		return "", fmt.Errorf("holds %d item mods; data.itemMods did not load", len(rows))
	}
	if !sort.SliceIsSorted(rows, func(i, j int) bool {
		return rows[i].Category < rows[j].Category || rows[i].Category == rows[j].Category && rows[i].ID < rows[j].ID
	}) {
		return "", fmt.Errorf("the item mods are not sorted by category and id; ItemModsOf binary-searches them")
	}
	var b strings.Builder
	p2eHeader(&b, "item-mods", "src/Modules/Data.lua:692-717")
	fmt.Fprintf(&b, "// ItemMods is data.itemMods' affix tables flattened to one row per mod, %d rows sorted by category and id.\nvar ItemMods = []ItemMod{\n", len(rows))
	for _, r := range rows {
		if len(r.Lines) == 0 || len(r.Lines) != len(r.StatOrder) {
			return "", fmt.Errorf("%s/%s has %d lines and %d stat orders", r.Category, r.ID, len(r.Lines), len(r.StatOrder))
		}
		nodeType, err := p2ePresent(r.ID, "nodeType", r.NodeType)
		if err != nil {
			return "", err
		}
		f := &p2eFields{}
		f.str("Category", r.Category)
		f.str("ID", r.ID)
		f.str("Type", r.Type)
		f.str("Affix", r.Affix)
		f.raw("Lines", p2eStrings(r.Lines))
		f.raw("StatOrder", p2eFloats(r.StatOrder))
		f.num("Level", r.Level)
		f.str("Group", r.Group)
		if r.WeightKey != nil {
			f.raw("WeightKey", p2eStrings(r.WeightKey))
			f.raw("WeightVal", p2eFloats(r.WeightVal))
		}
		if r.WeightMultiplierKey != nil {
			f.raw("WeightMultiplierKey", p2eStrings(r.WeightMultiplierKey))
			f.raw("WeightMultiplierVal", p2eFloats(r.WeightMultiplierVal))
		}
		if r.Tags != nil {
			f.raw("Tags", p2eStrings(r.Tags))
		}
		if r.ModTags != nil {
			f.raw("ModTags", p2eStrings(r.ModTags))
		}
		f.flag("Unscalable", r.Unscalable)
		f.num("NodeType", nodeType)
		b.WriteString("\t" + f.String() + ",\n")
	}
	b.WriteString("}\n")
	return b.String(), nil
}

type p2eModScalabilityJSON struct {
	Line    string `json:"line"`
	Entries []struct {
		IsScalable bool     `json:"isScalable"`
		Formats    []string `json:"formats"`
	} `json:"entries"`
}

// p2eModScalabilitySeparators are the bytes the row encoding splits on; no key or format may hold one.
const p2eModScalabilitySeparators = "\x1c\x1d\x1f"

func p2eModScalability(body []byte) (string, error) {
	var rows []p2eModScalabilityJSON
	if err := p2eDecode(body, &rows); err != nil {
		return "", err
	}
	if len(rows) < 1000 {
		return "", fmt.Errorf("holds %d modScalability rows; data.modScalability did not load", len(rows))
	}
	var b strings.Builder
	p2eHeader(&b, "mod-scalability", "src/Modules/Data.lua:481-481")
	fmt.Fprintf(&b, "// modScalabilityRows is data.modScalability, %d rows sorted by key: the key, U+001F, then its entries\n", len(rows))
	b.WriteString("// separated by U+001D. An entry is '1' when isScalable ('0' otherwise), then each format behind a U+001C.\n")
	b.WriteString("var modScalabilityRows = [...]string{\n")
	for i, r := range rows {
		if i > 0 && rows[i-1].Line >= r.Line {
			return "", fmt.Errorf("the modScalability rows are not sorted by key at %q", r.Line)
		}
		if strings.ContainsAny(r.Line, p2eModScalabilitySeparators) {
			return "", fmt.Errorf("modScalability key %q holds a separator byte of the row encoding", r.Line)
		}
		parts := make([]string, len(r.Entries))
		for j, e := range r.Entries {
			parts[j] = "0"
			if e.IsScalable {
				parts[j] = "1"
			}
			for _, f := range e.Formats {
				if f == "" || strings.ContainsAny(f, p2eModScalabilitySeparators) {
					return "", fmt.Errorf("modScalability[%q][%d] has a format %q the row encoding cannot hold", r.Line, j+1, f)
				}
				parts[j] += "\x1c" + f
			}
		}
		b.WriteString("\t" + strconv.Quote(r.Line+"\x1f"+strings.Join(parts, "\x1d")) + ",\n")
	}
	b.WriteString("}\n")
	return b.String(), nil
}

type p2eExclusiveModJSON struct {
	ID        string    `json:"id"`
	Lines     []string  `json:"lines"`
	StatOrder []float64 `json:"statOrder"`
}

func p2eItemModsExclusive(body []byte) (string, error) {
	var rows []p2eExclusiveModJSON
	if err := p2eDecode(body, &rows); err != nil {
		return "", err
	}
	if len(rows) < 1000 {
		return "", fmt.Errorf("holds %d exclusive mods; data.itemMods.Exclusive did not load", len(rows))
	}
	var b strings.Builder
	p2eHeader(&b, "item-mods-exclusive", "src/Modules/Data.lua:700-700")
	fmt.Fprintf(&b, "// ExclusiveMods is data.itemMods.Exclusive's lines and stat orders, %d rows sorted by id.\nvar ExclusiveMods = []ExclusiveMod{\n", len(rows))
	for i, r := range rows {
		if i > 0 && rows[i-1].ID >= r.ID {
			return "", fmt.Errorf("the exclusive mods are not sorted by id at %s", r.ID)
		}
		if len(r.Lines) != len(r.StatOrder) {
			return "", fmt.Errorf("%s has %d lines and %d stat orders", r.ID, len(r.Lines), len(r.StatOrder))
		}
		f := &p2eFields{}
		f.str("ID", r.ID)
		f.raw("Lines", p2eStrings(r.Lines))
		f.raw("StatOrder", p2eFloats(r.StatOrder))
		b.WriteString("\t" + f.String() + ",\n")
	}
	b.WriteString("}\n")
	return b.String(), nil
}
