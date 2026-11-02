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
