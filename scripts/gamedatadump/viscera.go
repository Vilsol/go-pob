package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
)

var visceraNumeralRe = regexp.MustCompile(`(\d+)$`)

// dumpViscera is Data/Viscera.lua: one item text per VisceraMod row, in Mods order.
func dumpViscera() ([]string, error) {
	//pob:block:src/Export/Scripts/viscera.lua:1-39
	if _, err := newModsExport(); err != nil {
		return nil, err
	}
	if err := poe.InitializeBaseItemTypes(gameDataContext(), raw.LatestVersion, assetCache()); err != nil {
		return nil, fmt.Errorf("BaseItemTypes: %w", err)
	}
	describer, err := loadStatFile("stat_descriptions")
	if err != nil {
		return nil, err
	}
	x := &modsExport{describer: describer}
	baseTypes := map[string]*poe.BaseItemType{}
	for _, b := range poe.BaseItemTypes {
		baseTypes[b.ID] = b
	}
	prefixMap := map[string]string{
		"viscera_low":  "Lesser ",
		"viscera_high": "Greater ",
	}
	var out []string
	for _, mod := range poe.Mods {
		if !strings.HasPrefix(mod.ID, "VisceraMod") {
			continue
		}
		prefix := ""
		for _, tag := range mod.TagsKeys {
			if p, ok := prefixMap[poe.Tags[tag].ID]; ok {
				prefix = p
				break
			}
		}
		numeral := visceraNumeralRe.FindString(mod.ID)
		id := "Metadata/Items/Viscera/Viscera" + strings.ReplaceAll(strings.TrimPrefix(strings.TrimPrefix(modFamily(mod, 0), "VisceraMod"), "Viscera"), " ", "")
		baseType := baseTypes[id]
		if baseType == nil {
			return nil, fmt.Errorf("could not find base type for %s", id)
		}
		var b strings.Builder
		b.WriteString(prefix + baseType.Name)
		if numeral != "" {
			b.WriteString(" " + numeral)
		}
		b.WriteString("\n" + baseType.Name)
		stats, _, err := x.describeMod(mod, nil)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", mod.ID, err)
		}
		b.WriteString("\nImplicits: " + strconv.Itoa(len(stats)))
		for _, stat := range stats {
			b.WriteString("\n" + stat)
		}
		b.WriteString("\n")
		out = append(out, b.String())
	}
	return out, nil
}
