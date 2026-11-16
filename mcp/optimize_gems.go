package mcp

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Vilsol/go-pob/pob"
)

// gemPair is one level and quality a gem slot can take.
type gemPair struct{ level, quality int }

// gemQualityMax is the quality Gemcutter's Prisms raise a gem to; corruption adds up to 3 more, the 23 PoB caps its
// default gem quality at (SkillsTab.lua:145, 536).
const (
	gemQualityMax          = 20
	gemQualityCorruptedMax = 23
)

// legalGem reports whether a gem can hold level and quality: up to its natural maximum level and 20 quality, or,
// corrupted, one level more (not an Awakened gem, SkillsTab.lua:1136-1141) or quality up to 23 (SkillsTab.lua:145),
// never both from one corruption.
//
//pob:block:src/Classes/SkillsTab.lua:1129-1143
func legalGem(p gemPair, natural int, awakened, corrupted bool) bool {
	if p.level < 1 || p.quality < 0 {
		return false
	}
	if p.level <= natural && p.quality <= gemQualityMax {
		return true
	}
	if !corrupted {
		return false
	}
	return (p.level == natural+1 && !awakened && p.quality <= gemQualityMax) || (p.level <= natural && p.quality <= gemQualityCorruptedMax)
}

// gemSlot varies one gem's level and quality over the listed values, keeping the pairs the gem can legally hold; the
// current pair is always a choice.
func gemSlot(b *pob.PathOfBuilding, in optSlotIn) (optSlot, *ToolError) {
	group, e := mainGroup(b, in.Group)
	if e != nil {
		return optSlot{}, e
	}
	g := (*groupsOf(b))[group-1]
	if in.Gem == "" {
		return optSlot{}, invalidArg("A gem slot needs gem.", nil)
	}
	at := slices.IndexFunc(g.Gems, func(x pob.Gem) bool { return gemMatches(x, in.Gem) })
	if at < 0 {
		return optSlot{}, gemNotInGroup(&g, in.Gem, group)
	}
	gem := g.Gems[at]
	natural := 20
	if ge, e := resolveGem(gem.NameSpec); e == nil && ge.MaxLevel > 0 {
		natural = ge.MaxLevel
	}
	awakened := strings.HasPrefix(gem.NameSpec, "Awakened ")
	maxLevel, maxQuality := natural, gemQualityMax
	if in.Corrupted {
		maxQuality = gemQualityCorruptedMax
		if !awakened {
			maxLevel = natural + 1
		}
	}
	for _, l := range in.Levels {
		if l < 1 || l > maxLevel {
			e := errf(CodeInvalidValue, "%s cannot be level %d%s.", gem.NameSpec, l, map[bool]string{true: "", false: " uncorrupted"}[in.Corrupted])
			e.Allowed = []any{fmt.Sprintf("1-%d", maxLevel)}
			return optSlot{}, e
		}
	}
	for _, q := range in.Qualities {
		if q < 0 || q > maxQuality {
			e := errf(CodeInvalidValue, "%s cannot have %d%% quality%s.", gem.NameSpec, q, map[bool]string{true: "", false: " uncorrupted"}[in.Corrupted])
			e.Allowed = []any{fmt.Sprintf("0-%d", maxQuality)}
			return optSlot{}, e
		}
	}
	// the gem's own level and quality stay a choice even when it could not hold them
	levels, qualities := in.Levels, in.Qualities
	if len(levels) == 0 {
		levels = []int{gem.Level, maxLevel}
	}
	if len(qualities) == 0 {
		qualities = []int{gem.Quality, maxQuality}
	}
	pairs := []gemPair{{gem.Level, gem.Quality}}
	for _, l := range levels {
		for _, q := range qualities {
			if p := (gemPair{l, q}); !slices.Contains(pairs, p) && legalGem(p, natural, awakened, in.Corrupted) {
				pairs = append(pairs, p)
			}
		}
	}
	label := func(p gemPair) string {
		s := fmt.Sprintf("%s level %d quality %d", gem.NameSpec, p.level, p.quality)
		switch {
		case p.quality > gemQualityCorruptedMax:
			s += fmt.Sprintf(" (above the game's %d%% maximum)", gemQualityCorruptedMax)
		case !legalGem(p, natural, awakened, false):
			s += " (corrupted)"
		}
		return s
	}
	return optSlot{
		name: fmt.Sprintf("gem g%d %s", group, gem.NameSpec), kind: slotOne, n: len(pairs), start: []int{0},
		apply: func(c *pob.PathOfBuilding, sel []int) {
			gg := &(*groupsOf(c))[group-1]
			gg.Gems = slices.Clone(gg.Gems)
			gg.Gems[at].Level, gg.Gems[at].Quality = pairs[sel[0]].level, pairs[sel[0]].quality
		},
		label:   func(sel []int) any { return label(pairs[sel[0]]) },
		sizeMsg: fmt.Sprintf("%d level and quality pairs", len(pairs)),
		moveMsg: fmt.Sprintf("%d level and quality pairs", len(pairs)),
	}, nil
}
