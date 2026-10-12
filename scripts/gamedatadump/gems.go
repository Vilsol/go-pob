package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
)

type gemRow struct {
	ID                       string   `json:"id"`
	GameID                   string   `json:"gameId"`
	Loc                      int      `json:"loc"`
	Name                     string   `json:"name"`
	Tags                     []string `json:"tags"`
	GrantedEffectID          string   `json:"grantedEffectId"`
	SecondaryGrantedEffectID string   `json:"secondaryGrantedEffectId,omitempty"`
	ReqStr                   int      `json:"reqStr"`
	ReqDex                   int      `json:"reqDex"`
	ReqInt                   int      `json:"reqInt"`
	NaturalMaxLevel          int      `json:"naturalMaxLevel"`
}

// This is here to fix name collisions like in the case of Barrage
//
//pob:block:src/Export/Scripts/skills.lua:6-9
var fullNameGems = map[string]bool{
	"Metadata/Items/Gems/SupportGemBarrage": true,
}

//pob:block:src/Export/Scripts/skills.lua:542-544
var skillTemplates = []string{"act_str", "act_dex", "act_int", "other", "glove", "minion", "spectre", "sup_str", "sup_dex", "sup_int"}

var (
	templateDirectiveRe = regexp.MustCompile(`#(\pL+) ?(.*)`)
	skillDirectiveRe    = regexp.MustCompile(`^([\pL\pN]+) (.+)`)
)

//pob:block:src/Export/Scripts/skills.lua:60-146
func gemTemplates(pobDir string) (map[string]bool, map[string]string, error) {
	gemEffectOf := func(granted int) (*poe.GemEffect, bool) {
		for _, g := range poe.GemEffects {
			if g.GrantedEffect == granted {
				return g, false
			}
		}
		for _, g := range poe.GemEffects {
			if g.GrantedEffect2 != nil && *g.GrantedEffect2 == granted {
				return g, true
			}
		}
		return nil, false
	}
	gems, trueGemNames := map[string]bool{}, map[string]string{}
	for _, name := range skillTemplates {
		f, err := os.Open(filepath.Join(pobDir, "src", "Export", "Skills", name+".txt")) //nolint:gosec // the pinned PoB checkout
		if err != nil {
			return nil, nil, fmt.Errorf("template %s: %w", name, err)
		}
		noGem := false
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			m := templateDirectiveRe.FindStringSubmatch(sc.Text())
			if m == nil {
				continue
			}
			switch m[1] {
			case "noGem":
				noGem = true
			case "skill":
				grantedID := m[2]
				if s := skillDirectiveRe.FindStringSubmatch(m[2]); s != nil {
					grantedID = s[1]
				}
				granted := poe.GrantedEffectByID(grantedID)
				if granted == nil {
					continue
				}
				gemEffect, _ := gemEffectOf(granted.Key)
				var skillGem *poe.SkillGem
				if gemEffect != nil {
				gems:
					for _, gem := range poe.SkillGems {
						for _, variant := range gem.GemVariants {
							if poe.GemEffects[variant].ID == gemEffect.ID {
								skillGem = gem
								for _, g := range poe.GemEffects {
									if g.ID == gemEffect.ID {
										if g.Name != "" {
											trueGemNames[gemEffect.ID] = g.Name
										}
										break
									}
								}
								break gems
							}
						}
					}
				}
				if skillGem != nil && !noGem {
					gems[gemEffect.ID] = true
				}
				noGem = false
			}
		}
		_ = f.Close()
		if err := sc.Err(); err != nil {
			return nil, nil, fmt.Errorf("template %s: %w", name, err)
		}
	}
	return gems, trueGemNames, nil
}

//pob:block:src/Export/Scripts/skills.lua:546-592
func dumpGems() (any, error) {
	pobDir := os.Getenv("POB_DIR")
	if pobDir == "" {
		return nil, fmt.Errorf("POB_DIR is not set; the gem set is the pinned PoB's Export/Skills templates")
	}
	ctx, cache := gameDataContext(), assetCache()
	for _, t := range []struct {
		name string
		init func() error
	}{
		{"BaseItemTypes", func() error { return poe.InitializeBaseItemTypes(ctx, raw.LatestVersion, cache) }},
		{"ActiveSkills", func() error { return poe.InitializeActiveSkills(ctx, raw.LatestVersion, cache) }},
		{"GrantedEffects", func() error { return poe.InitializeGrantedEffects(ctx, raw.LatestVersion, cache) }},
		{"GemTags", func() error { return poe.InitializeGemTags(ctx, raw.LatestVersion, cache) }},
		{"GemEffects", func() error { return poe.InitializeGemEffects(ctx, raw.LatestVersion, cache) }},
		{"SkillGems", func() error { return poe.InitializeSkillGems(ctx, raw.LatestVersion, cache) }},
		{"ItemExperiencePerLevel", func() error { return poe.InitializeItemExperiencePerLevels(ctx, raw.LatestVersion, cache) }},
	} {
		if err := t.init(); err != nil {
			return nil, fmt.Errorf("%s: %w", t.name, err)
		}
	}
	gems, trueGemNames, err := gemTemplates(pobDir)
	if err != nil {
		return nil, err
	}
	if len(gems) == 0 {
		return nil, fmt.Errorf("the templates give no gem")
	}

	out := map[string]gemRow{}
	tagNames := map[string]bool{}
	line := 4
	for _, skillGem := range poe.SkillGems {
		base := poe.BaseItemTypes[skillGem.BaseItemType]
		for _, gemEffect := range skillGem.GetGemVariants() {
			if !gems[gemEffect.ID] {
				continue
			}
			// Some skills have an additional old version that exists in the game files and messes up transfigured gem matching
			delete(gems, gemEffect.ID)
			line++
			id := "Metadata/Items/Gems/SkillGem" + gemEffect.ID
			name := strings.ReplaceAll(base.Name, " Support", "")
			if fullNameGems[base.ID] {
				name = base.Name
			} else if n, ok := trueGemNames[gemEffect.ID]; ok {
				name = n
			}
			granted := poe.GrantedEffects[gemEffect.GrantedEffect]
			row := gemRow{ID: id, GameID: base.ID, Loc: line, Name: name, GrantedEffectID: granted.ID,
				ReqStr: skillGem.Str, ReqDex: skillGem.Dex, ReqInt: skillGem.Int}
			line++
			if !skillGem.IsSupport {
				line++
			}
			line += 3
			if gemEffect.GrantedEffect2 != nil {
				row.SecondaryGrantedEffectID = poe.GrantedEffects[*gemEffect.GrantedEffect2].ID
				line++
			}
			if gemEffect.SupportName != "" {
				line++
			}
			if skillGem.IsVaalGem {
				line++
			}
			row.Tags = []string{}
			for _, t := range gemEffect.GemTags {
				row.Tags = append(row.Tags, poe.GemTags[t].ID)
				tagNames[poe.GemTags[t].ID] = true
			}
			line += 2 + len(gemEffect.GemTags)
			line += 4
			row.NaturalMaxLevel = max(len(skillGem.GetLevelProgression()), 1)
			line += 2
			sort.Strings(row.Tags)
			out[id] = row
		}
	}
	sortedTags := make([]string, 0, len(tagNames))
	for t := range tagNames {
		sortedTags = append(sortedTags, t)
	}
	sort.Strings(sortedTags)
	return map[string]any{"pobVersion": "go-pob-data " + raw.LatestVersion, "gemCount": len(out), "tagNames": sortedTags, "gems": out}, nil
}
