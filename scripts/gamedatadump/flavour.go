package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
)

const luaSpace = " \t\n\v\f\r"

var (
	flavourBracesRe  = regexp.MustCompile(`\{(.*?)\}`)
	flavourAnglesRe  = regexp.MustCompile(`<<(.*?)>>`)
	flavourTrailUndR = regexp.MustCompile(`_+$`)
)

//pob:block:src/Export/Scripts/flavourText.lua:10-48
func cleanAndSplit(str string) []string {
	str = strings.ReplaceAll(str, "\r\n", "\n")
	str = strings.ReplaceAll(str, "<default>", "\n^8")
	lines := []string{}
	for _, line := range strings.Split(str, "\n") {
		if line == "" {
			continue
		}
		line = strings.Trim(line, luaSpace)
		if line == "" {
			continue
		}
		line = flavourBracesRe.ReplaceAllString(line, "$1")
		line = flavourAnglesRe.ReplaceAllString(line, "")
		line = strings.Trim(line, luaSpace)
		if strings.HasPrefix(line, "^8") && (len(lines) == 0 || lines[len(lines)-1] != "") {
			lines = append(lines, "")
		}
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

type flavourUnique struct {
	Name string   `json:"name"`
	ID   string   `json:"id"`
	Text []string `json:"text"`
}

//pob:block:src/Export/Scripts/flavourText.lua:4-8
//pob:block:src/Export/Scripts/flavourText.lua:123-197
func dumpFlavourText() (any, error) {
	pobDir := os.Getenv("POB_DIR")
	if pobDir == "" {
		return nil, fmt.Errorf("POB_DIR is not set; the forced names and base templates are the pinned PoB's")
	}
	ctx, cache := gameDataContext(), assetCache()
	for _, t := range []struct {
		name string
		init func() error
	}{
		{"Words", func() error { return poe.InitializeWords(ctx, raw.LatestVersion, cache) }},
		{"ItemVisualIdentity", func() error { return poe.InitializeItemVisualIdentities(ctx, raw.LatestVersion, cache) }},
		{"UniqueStashLayout", func() error { return poe.InitializeUniqueStashLayouts(ctx, raw.LatestVersion, cache) }},
		{"FlavourText", func() error { return poe.InitializeFlavourTexts(ctx, raw.LatestVersion, cache) }},
		{"Stats", func() error { return poe.InitializeStats(ctx, raw.LatestVersion, cache) }},
		{"Tags", func() error { return poe.InitializeTags(ctx, raw.LatestVersion, cache) }},
		{"ModType", func() error { return poe.InitializeModTypes(ctx, raw.LatestVersion, cache) }},
		{"BuffTemplates", func() error { return poe.InitializeBuffTemplates(ctx, raw.LatestVersion, cache) }},
		{"Mods", func() error { return poe.InitializeMods(ctx, raw.LatestVersion, cache) }},
	} {
		if err := t.init(); err != nil {
			return nil, fmt.Errorf("%s: %w", t.name, err)
		}
	}
	script, err := os.ReadFile(filepath.Join(pobDir, "src", "Export", "Scripts", "flavourText.lua")) //nolint:gosec // the pinned PoB checkout
	if err != nil {
		return nil, fmt.Errorf("flavourText.lua: %w", err)
	}
	forced, err := luaTableLiteral(string(script), "local forcedNameList = ")
	if err != nil {
		return nil, fmt.Errorf("flavourText.lua: %w", err)
	}

	uniqueNameLookup := map[string]string{}
	for _, row := range poe.UniqueStashLayouts {
		name := poe.Words[row.WordsKey].Text2
		id := flavourTrailUndR.ReplaceAllString(poe.ItemVisualIdentities[row.ItemVisualIdentityKey].ID, "")
		if strings.Contains(id, "Map") || strings.Contains(id, "AlternateArt") || strings.Contains(id, "AtlasUpgrade") || strings.Contains(id, "HeistQuest") {
			continue
		}
		uniqueNameLookup[id] = name
	}
	flavourTextByID := map[string][]string{}
	for _, c := range poe.FlavourTexts {
		flavourTextByID[flavourTrailUndR.ReplaceAllString(c.ID, "")] = cleanAndSplit(c.Text)
	}

	uniques := []flavourUnique{}
	for _, e := range forced.arr {
		entry, ok := e.(*luaTable)
		if !ok {
			return nil, fmt.Errorf("flavourText.lua: a forcedNameList entry is not a table")
		}
		id, _ := entry.hash["id"].(string)
		name, _ := entry.hash["name"].(string)
		if lines, ok := flavourTextByID[id]; ok {
			uniques = append(uniques, flavourUnique{Name: name, ID: id, Text: lines})
		}
	}
	for _, id := range sortedKeys(uniqueNameLookup) {
		if lines, ok := flavourTextByID[id]; ok {
			uniques = append(uniques, flavourUnique{Name: uniqueNameLookup[id], ID: id, Text: lines})
		}
	}

	bases, err := dumpItemBases(pobDir)
	if err != nil {
		return nil, err
	}
	baseText := map[string][]string{}
	for name, b := range bases {
		if len(b.FlavourText) > 0 {
			baseText[name] = b.FlavourText
		}
	}
	return map[string]any{"pobVersion": "go-pob-data " + raw.LatestVersion, "uniques": uniques, "bases": baseText}, nil
}

func luaTableLiteral(src, marker string) (*luaTable, error) {
	_, rest, ok := strings.Cut(src, marker)
	if !ok {
		return nil, fmt.Errorf("no %q", marker)
	}
	depth, end := 0, -1
	for i, c := range rest {
		if c == '{' {
			depth++
		} else if c == '}' {
			if depth--; depth == 0 {
				end = i + 1
				break
			}
		}
	}
	if end < 0 {
		return nil, fmt.Errorf("%q is not a closed table", marker)
	}
	toks, err := luaLex(rest[:end])
	if err != nil {
		return nil, fmt.Errorf("%q: %w", marker, err)
	}
	v, err := (&luaParser{toks: toks}).expr()
	if err != nil {
		return nil, fmt.Errorf("%q: %w", marker, err)
	}
	t, ok := v.(*luaTable)
	if !ok {
		return nil, fmt.Errorf("%q is not a table", marker)
	}
	return t, nil
}
