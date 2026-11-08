package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob-data/poe2/pobdata"
	"github.com/tinylib/msgp/msgp"
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
		note:     "data.itemMods' affix tables (Item, Flask, Charm, Jewel, IncursionLimb, Corruption, Desecrated) after Data.lua's load (src/Modules/Data.lua:692-717); tradeHashes dropped. Game data FROM go-pob-data: `gamedatadump poe2-mods` runs PoB-PoE2's Export/Scripts/mods.lua over 0.5's Mods, Stats, ModType, ModFamily and Tags tables and stat descriptions, every row equal to the pin's src/Data/Mod*.lua up to the order of each tradeHashes table (`mise run poe2-gamedata-check`), and regendata_poe2.lua loads them in place of the pin's (GAMEDATA_DUMP). Published as go-pob-data's 0.5 pob/ItemMods asset, which the generated file names; GO_POB_DATA_DIR=<go-pob-data checkout> writes it there",
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
		note:     "data.modScalability (src/Data/ModScalability.lua, src/Modules/Data.lua:481), the table itemLib.applyRange reads first. Game data FROM go-pob-data: `gamedatadump poe2-mod-scalability` runs PoB-PoE2's Export/Scripts/modScalability.lua over 0.5's Stats table and stat descriptions, every row equal to the pin's up to one later wording go-pob-data carries (`mise run poe2-gamedata-check`), and regendata_poe2.lua loads it in place of the pin's (GAMEDATA_DUMP). Published as go-pob-data's 0.5 pob/ModScalability asset, which the generated file names; GO_POB_DATA_DIR=<go-pob-data checkout> writes it there",
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
		note:     "data.itemMods.Exclusive's lines and stat orders (src/Modules/Data.lua:700), which Item.lua orders an advanced-copy unique's lines by. Game data FROM go-pob-data: `gamedatadump poe2-mods` writes src/Data/ModItemExclusive.lua, every row equal to the pin's up to the order of each tradeHashes table (`mise run poe2-gamedata-check`), and regendata_poe2.lua loads it in place of the pin's (GAMEDATA_DUMP). Published as go-pob-data's 0.5 pob/ExclusiveMods asset, which the generated file names; GO_POB_DATA_DIR=<go-pob-data checkout> writes it there",
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

func p2eItemMods(body []byte) (string, *p2eAsset, error) {
	var rows []p2eItemModJSON
	if err := p2eDecode(body, &rows); err != nil {
		return "", nil, err
	}
	if len(rows) < 1000 {
		return "", nil, fmt.Errorf("holds %d item mods; data.itemMods did not load", len(rows))
	}
	if !sort.SliceIsSorted(rows, func(i, j int) bool {
		return rows[i].Category < rows[j].Category || rows[i].Category == rows[j].Category && rows[i].ID < rows[j].ID
	}) {
		return "", nil, fmt.Errorf("the item mods are not sorted by category and id; ItemModsOf binary-searches them")
	}
	mods := make([]pobdata.ItemMod, len(rows))
	for i, r := range rows {
		if len(r.Lines) == 0 || len(r.Lines) != len(r.StatOrder) {
			return "", nil, fmt.Errorf("%s/%s has %d lines and %d stat orders", r.Category, r.ID, len(r.Lines), len(r.StatOrder))
		}
		nodeType, err := p2ePresent(r.ID, "nodeType", r.NodeType)
		if err != nil {
			return "", nil, err
		}
		mods[i] = pobdata.ItemMod{Category: r.Category, ID: r.ID, Type: r.Type, Affix: r.Affix, Lines: r.Lines,
			StatOrder: r.StatOrder, Level: r.Level, Group: r.Group, Tags: r.Tags, ModTags: r.ModTags,
			Unscalable: r.Unscalable, NodeType: nodeType}
		if r.WeightKey != nil {
			mods[i].WeightKey, mods[i].WeightVal = r.WeightKey, r.WeightVal
		}
		if r.WeightMultiplierKey != nil {
			mods[i].WeightMultiplierKey, mods[i].WeightMultiplierVal = r.WeightMultiplierKey, r.WeightMultiplierVal
		}
	}
	return p2eAssetSource("item-mods", "src/Modules/Data.lua:692-717", pobdata.ItemModsTable, "itemModsAsset",
		fmt.Sprintf("data.itemMods' affix tables flattened to one row per mod, %d rows sorted by category and id", len(rows)), mods)
}

type p2eModScalabilityJSON struct {
	Line    string `json:"line"`
	Entries []struct {
		IsScalable bool     `json:"isScalable"`
		Formats    []string `json:"formats"`
	} `json:"entries"`
}

func p2eModScalability(body []byte) (string, *p2eAsset, error) {
	var rows []p2eModScalabilityJSON
	if err := p2eDecode(body, &rows); err != nil {
		return "", nil, err
	}
	if len(rows) < 1000 {
		return "", nil, fmt.Errorf("holds %d modScalability rows; data.modScalability did not load", len(rows))
	}
	out := make([]pobdata.ModScalability, len(rows))
	for i, r := range rows {
		if i > 0 && rows[i-1].Line >= r.Line {
			return "", nil, fmt.Errorf("the modScalability rows are not sorted by key at %q", r.Line)
		}
		out[i].Line = r.Line
		if len(r.Entries) > 0 {
			out[i].Entries = make([]pobdata.ModScalabilityEntry, len(r.Entries))
		}
		for j, e := range r.Entries {
			out[i].Entries[j].IsScalable = e.IsScalable
			if len(e.Formats) > 0 {
				out[i].Entries[j].Formats = e.Formats
			}
		}
	}
	return p2eAssetSource("mod-scalability", "src/Modules/Data.lua:481-481", pobdata.ModScalabilityTable, "modScalabilityAsset",
		fmt.Sprintf("data.modScalability, %d rows sorted by key", len(rows)), out)
}

type p2eExclusiveModJSON struct {
	ID        string    `json:"id"`
	Lines     []string  `json:"lines"`
	StatOrder []float64 `json:"statOrder"`
}

func p2eItemModsExclusive(body []byte) (string, *p2eAsset, error) {
	var rows []p2eExclusiveModJSON
	if err := p2eDecode(body, &rows); err != nil {
		return "", nil, err
	}
	if len(rows) < 1000 {
		return "", nil, fmt.Errorf("holds %d exclusive mods; data.itemMods.Exclusive did not load", len(rows))
	}
	mods := make([]pobdata.ExclusiveMod, len(rows))
	for i, r := range rows {
		if i > 0 && rows[i-1].ID >= r.ID {
			return "", nil, fmt.Errorf("the exclusive mods are not sorted by id at %s", r.ID)
		}
		if len(r.Lines) != len(r.StatOrder) {
			return "", nil, fmt.Errorf("%s has %d lines and %d stat orders", r.ID, len(r.Lines), len(r.StatOrder))
		}
		mods[i] = pobdata.ExclusiveMod{ID: r.ID, Lines: r.Lines, StatOrder: r.StatOrder}
	}
	return p2eAssetSource("item-mods-exclusive", "src/Modules/Data.lua:700-700", pobdata.ExclusiveModsTable, "exclusiveModsAsset",
		fmt.Sprintf("data.itemMods.Exclusive's lines and stat orders, %d rows sorted by id", len(rows)), mods)
}

// p2eAssetVersion is the go-pob-data version the PoE2 tables published as go-pob-data assets belong to.
const p2eAssetVersion = "0.5"

// p2eAsset is a table go-pob-data publishes under pob/, named by its contents: its bytes are written into
// the go-pob-data checkout at $GO_POB_DATA_DIR when that is set, and the generated Go names it.
type p2eAsset struct {
	path    string
	encoded []byte
}

func (a *p2eAsset) write() error {
	dir := os.Getenv("GO_POB_DATA_DIR")
	if dir == "" {
		return nil
	}
	compressed, err := pobdata.Compress(a.encoded)
	if err != nil {
		return fmt.Errorf("could not compress %s: %w", a.path, err)
	}
	out := filepath.Join(dir, "data", p2eAssetVersion, filepath.FromSlash(a.path))
	if err := os.MkdirAll(filepath.Dir(out), 0o750); err != nil { //nolint:gosec // the caller's go-pob-data checkout
		return fmt.Errorf("could not create %s: %s", filepath.Dir(out), errnoText(err))
	}
	if err := os.WriteFile(out, compressed, 0o644); err != nil { //nolint:gosec // a published asset
		return fmt.Errorf("could not write %s: %s", out, errnoText(err))
	}
	return nil
}

func p2eAssetSource[T any, PT interface {
	*T
	msgp.Encodable
}](table, cite, assetTable, goName, doc string, rows []T) (string, *p2eAsset, error) {
	encoded, err := pobdata.Encode[T, PT](rows)
	if err != nil {
		return "", nil, err
	}
	asset := &p2eAsset{path: pobdata.AssetPath(assetTable, encoded), encoded: encoded}
	var b strings.Builder
	p2eHeader(&b, table, cite)
	fmt.Fprintf(&b, "// %s is go-pob-data's %s %s asset: %s.\n", goName, p2eAssetVersion, assetTable, doc)
	b.WriteString("// Regenerating with GO_POB_DATA_DIR set to a go-pob-data checkout writes the asset into it.\n")
	fmt.Fprintf(&b, "const %s = %s\n", goName, strconv.Quote(asset.path))
	return b.String(), asset, nil
}
