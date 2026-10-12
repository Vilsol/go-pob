package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Vilsol/go-pob-data/loader"
	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
)

type craftBase struct {
	Implicit         string            `json:"implicit"`
	Hidden           bool              `json:"hidden"`
	CannotBeAnointed bool              `json:"cannotBeAnointed"`
	Tags             []string          `json:"tags"`
	InfluenceTags    map[string]string `json:"influenceTags,omitempty"`
	ImplicitModTypes [][]string        `json:"implicitModTypes"`
	Type             string            `json:"type"`
	SubType          string            `json:"subType"`
	FlavourText      []string          `json:"-"`
}

//pob:block:src/Export/Scripts/bases.lua:419-442
var baseItemTypes = []string{"axe", "bow", "claw", "dagger", "fishing", "mace", "staff", "sword", "wand", "helmet", "body", "gloves", "boots",
	"shield", "quiver", "amulet", "ring", "belt", "jewel", "flask", "tincture", "graft"}

type baseState struct {
	typ, subType         string
	influenceBaseTag     *string
	forceShow, forceHide bool
}

type basesExport struct {
	x     *modsExport
	tags  map[string][]string
	bases map[string]craftBase
}

var (
	baseDirectiveRe    = regexp.MustCompile(`^([A-Za-z0-9/_]+) (.+)`)
	itExtendsRe        = regexp.MustCompile(`extends "(.+)"`)
	itRemoveTagRe      = regexp.MustCompile(`remove_tag = "(.+)"`)
	itTagRe            = regexp.MustCompile(`tag = "(.+)"`)
	baseInfluenceNames = []string{"shaper", "elder", "adjudicator", "basilisk", "crusader", "eyrie", "cleansing", "tangle"}
)

//pob:block:src/Export/Scripts/bases.lua:87-111
func (b *basesExport) baseItemTags(baseItemType string) []string {
	if baseItemType == "nothing" {
		return []string{}
	}
	if tags, ok := b.tags[baseItemType]; ok {
		return tags
	}
	file, err := loader.LoadFile(gameDataContext(), raw.LatestVersion, baseItemType+".it", assetCache())
	if err != nil {
		b.tags[baseItemType] = nil
		return nil
	}
	tags := []string{}
	for _, line := range strings.FieldsFunc(string(file), func(r rune) bool { return r == '\r' || r == '\n' }) {
		switch {
		case itExtendsRe.MatchString(line):
			tags = append(tags, b.baseItemTags(itExtendsRe.FindStringSubmatch(line)[1])...)
		case strings.Contains(line, "remove_tag"):
			i := len(tags) - 1
			if m := itRemoveTagRe.FindStringSubmatch(line); m != nil {
				for j, t := range tags {
					if t == m[1] {
						i = j
						break
					}
				}
			}
			if i >= 0 {
				tags = append(tags[:i], tags[i+1:]...)
			}
		case strings.Contains(line, "tag"):
			if m := itTagRe.FindStringSubmatch(line); m != nil {
				tags = append(tags, m[1])
			}
		}
	}
	b.tags[baseItemType] = tags
	return tags
}

//pob:block:src/Export/Scripts/bases.lua:77-203
func (b *basesExport) base(st *baseState, args string) error {
	baseTypeID, displayName := args, ""
	if m := baseDirectiveRe.FindStringSubmatch(args); m != nil {
		baseTypeID, displayName = m[1], m[2]
	}
	var baseItemType *poe.BaseItemType
	for _, t := range poe.BaseItemTypes {
		if t.ID == baseTypeID {
			baseItemType = t
			break
		}
	}
	if baseItemType == nil {
		fmt.Fprintf(os.Stderr, "Invalid Id %s\n", baseTypeID)
		return nil
	}
	baseItemTags := b.baseItemTags(baseItemType.InheritsFrom)
	if displayName == "" {
		displayName = baseItemType.Name
	}
	displayName = strings.TrimSpace(strings.ReplaceAll(displayName, "ö", "o"))
	if displayName == "Energy Blade" {
		if st.typ == "One Handed Sword" {
			displayName = "Energy Blade One Handed"
		} else {
			displayName = "Energy Blade Two Handed"
		}
	}
	out := craftBase{Type: st.typ, Hidden: st.forceHide && !strings.Contains(baseTypeID, "Talisman") && !st.forceShow, ImplicitModTypes: [][]string{}}
	out.SubType = st.subType
	combined := map[string]bool{}
	for _, t := range baseItemTags {
		combined[t] = true
	}
	for _, t := range baseItemType.TagsKeys {
		combined[poe.Tags[t].ID] = true
	}
	out.Tags = make([]string, 0, len(combined))
	for t := range combined {
		out.Tags = append(out.Tags, t)
	}
	sort.Strings(out.Tags)
	if st.influenceBaseTag != nil {
		out.InfluenceTags = map[string]string{}
		for _, s := range baseInfluenceNames {
			out.InfluenceTags[s] = *st.influenceBaseTag + "_" + s
		}
	}
	var implicitLines []string
	for _, m := range baseItemType.ImplicitModsKeys {
		mod := poe.Mods[m]
		lines, _, err := b.x.describeMod(mod, nil)
		if err != nil {
			return fmt.Errorf("%s: %w", mod.ID, err)
		}
		for _, line := range lines {
			implicitLines = append(implicitLines, line)
			out.ImplicitModTypes = append(out.ImplicitModTypes, modTags(mod.ImplicitTagsKeys))
		}
	}
	out.Implicit = strings.Join(implicitLines, "\n")
	out.CannotBeAnointed = len(baseItemType.EnchantMods) > 0
	//pob:block:src/Export/Scripts/bases.lua:307-316
	if k := baseItemType.FlavourTextKey; k != nil {
		out.FlavourText = cleanAndSplit(poe.FlavourTexts[*k].Text)
	}
	b.bases[displayName] = out
	return nil
}

func luaPatternRe(pattern string) (*regexp.Regexp, error) {
	re, err := regexp.Compile(strings.NewReplacer("%d", "[0-9]", "%w", "[A-Za-z0-9]", "%a", "[A-Za-z]", "%%", "%").Replace(pattern))
	if err != nil {
		return nil, fmt.Errorf("pattern %q: %w", pattern, err)
	}
	return re, nil
}

//pob:block:src/Export/Scripts/bases.lua:328-346
func (b *basesExport) baseMatch(st *baseState, argstr string) error {
	args := strings.Fields(argstr)
	if len(args) == 0 {
		return nil
	}
	key, value := "Id", args[0]
	if len(args) > 1 {
		key, value = args[0], args[1]
	}
	re, err := luaPatternRe(value)
	if err != nil {
		return err
	}
	for _, t := range poe.BaseItemTypes {
		col := t.ID
		switch key {
		case "Id":
		case "BaseType":
			col = t.InheritsFrom
		default:
			return fmt.Errorf("baseMatch column %s is not modelled", key)
		}
		if re.MatchString(col) && !strings.Contains(t.ID, "Royale") {
			if err := b.base(st, t.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

//pob:block:src/Export/Scripts/bases.lua:443-445
func dumpItemBases(pobDir string) (map[string]craftBase, error) {
	ctx, cache := gameDataContext(), assetCache()
	if err := poe.InitializeBaseItemTypes(ctx, raw.LatestVersion, cache); err != nil {
		return nil, fmt.Errorf("BaseItemTypes: %w", err)
	}
	if err := poe.InitializeFlavourTexts(ctx, raw.LatestVersion, cache); err != nil {
		return nil, fmt.Errorf("FlavourText: %w", err)
	}
	describer, err := loadStatFile("tincture_stat_descriptions")
	if err != nil {
		return nil, err
	}
	b := &basesExport{x: &modsExport{describer: describer}, tags: map[string][]string{}, bases: map[string]craftBase{}}
	for _, name := range baseItemTypes {
		text, err := os.ReadFile(filepath.Join(pobDir, "src", "Export", "Bases", name+".txt")) //nolint:gosec // the pinned PoB checkout
		if err != nil {
			return nil, fmt.Errorf("template %s: %w", name, err)
		}
		st := &baseState{}
		for _, line := range strings.Split(string(text), "\n") {
			m := templateLineRe.FindStringSubmatch(strings.TrimSuffix(line, "\r"))
			if m == nil {
				continue
			}
			args := m[2]
			switch m[1] {
			case "type":
				st.typ = args
			case "subType":
				st.subType = args
			case "influenceBaseTag":
				st.influenceBaseTag = &args
			case "forceShow":
				st.forceShow = args == "true"
			case "forceHide":
				st.forceHide = args == "true"
			case "socketLimit":
			case "base":
				err = b.base(st, args)
			case "baseMatch":
				err = b.baseMatch(st, args)
			default:
				fmt.Fprintf(os.Stderr, "Unknown directive '%s'\n", m[1])
			}
			if err != nil {
				return nil, err
			}
		}
	}
	return b.bases, nil
}
