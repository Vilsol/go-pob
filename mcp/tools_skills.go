package mcp

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
)

type gemEntry struct {
	Name      string
	GemID     string
	VariantID string
	SkillID   string
	Support   bool
	Tags      []string
	MaxLevel  int
	levelReq  int
}

var gemList []gemEntry

func gems() []gemEntry {
	if gemList != nil {
		return gemList
	}
	add := func(gemID, variant string, row raw.PoBGem) {
		skill := raw.PoBSkills[row.GrantedEffectID]
		if gemID == "" || skill.HideFromGemList || skill.Legacy {
			return
		}
		g := gemEntry{Name: row.Name, GemID: gemID, VariantID: variant, SkillID: row.GrantedEffectID, Support: row.Tags["support"], MaxLevel: row.NaturalMaxLevel, levelReq: -1}
		for t, on := range row.Tags {
			if on {
				g.Tags = append(g.Tags, t)
			}
		}
		sort.Strings(g.Tags)
		gemList = append(gemList, g)
	}
	for id, row := range raw.Gems {
		add(id, "", row)
	}
	for id, row := range raw.GemVariants {
		add(raw.GemGameIDForGrantedEffectID[id], id, row)
	}
	slices.SortFunc(gemList, func(a, b gemEntry) int { return strings.Compare(a.Name, b.Name) })
	return gemList
}

//pob:block:src/Classes/GemSelectControl.lua:126
func (g *gemEntry) LevelReq() int {
	if g.levelReq >= 0 {
		return g.levelReq
	}
	g.levelReq = 1
	if ge := poe.GrantedEffectByID(g.SkillID); ge != nil {
		if calc := raw.GetCalculatedGrantedEffect(ge); calc != nil {
			if first, ok := calc.GetCalculatedLevels()[1]; ok && first.LevelRequirement != 0 {
				g.levelReq = first.LevelRequirement
			}
		}
	}
	return g.levelReq
}

func gemNames(supportOnly *bool) []string {
	var out []string
	for _, g := range gems() {
		if supportOnly == nil || g.Support == *supportOnly {
			out = append(out, g.Name)
		}
	}
	return out
}

func resolveGem(name string) (gemEntry, *ToolError) {
	list := gems()
	g, ok, hits := resolveName(name, list, func(g gemEntry) string { return g.Name }, nil)
	if !ok && len(hits) == 0 {
		// "Fire Penetration Support" names the gem PoB calls "Fire Penetration"
		if trimmed, cut := strings.CutSuffix(norm(name), " support"); cut {
			g, ok, hits = resolveName(trimmed, list, func(g gemEntry) string { return g.Name }, nil)
		}
	}
	if ok {
		return g, nil
	}
	if len(hits) > 0 {
		e := errf(CodeAmbiguous, "%q matches several gems.", name)
		for _, h := range hits[:min(len(hits), 8)] {
			e.Candidates = append(e.Candidates, h.Name)
		}
		e.Hint = "Retry with the full gem name."
		return gemEntry{}, e
	}
	return gemEntry{}, notFound("gem", name, suggest(name, gemNames(nil), 5), "Retry with one of did_you_mean, or search_data kind=gem.")
}

func newGem(g gemEntry, level, quality int) pob.Gem {
	count := 1
	if g.MaxLevel > 0 {
		level = min(level, max(g.MaxLevel, 1))
	}
	gem := pob.Gem{
		GemID: g.GemID, NameSpec: g.Name, SkillID: g.SkillID, Level: level, Quality: quality, QualityID: "Default",
		Count: &count, Enabled: true, EnableGlobal1: true, EnableGlobal2: true,
	}
	if g.VariantID != "" {
		v := g.VariantID
		gem.VariantID = &v
	}
	return gem
}

func gemMatches(gem pob.Gem, name string) bool {
	return norm(gem.NameSpec) == norm(name) || norm(gem.NameSpec) == strings.TrimSuffix(norm(name), " support")
}

type skillOp struct {
	Op      string   `json:"op"`
	Group   int      `json:"group"`
	Slot    string   `json:"slot"`
	Gems    []string `json:"gems"`
	Gem     string   `json:"gem"`
	Level   *int     `json:"level"`
	Quality *int     `json:"quality"`
	FullDPS *bool    `json:"full_dps"`
}

type editSkillsIn struct {
	Handle string    `json:"handle"`
	DryRun bool      `json:"dry_run"`
	Watch  []string  `json:"watch"`
	Ops    []skillOp `json:"ops"`
}

var skillOps = []string{"add_group", "set_gems", "add_gem", "remove_gem", "set_main", "enable_group", "disable_group", "remove_group", "set_gem_level"}

func (s *Server) editSkills(args json.RawMessage) (any, *ToolError) {
	var in editSkillsIn
	if e := decode(args, schemaEditSkills, &in); e != nil {
		return nil, e
	}
	if e := checkOps(len(in.Ops), 10); e != nil {
		return nil, e
	}
	return s.runEdit(in.Handle, in.DryRun, in.Watch, len(in.Ops), func(ec *editCtx, i int) *ToolError {
		return applySkillOp(ec, in.Ops[i])
	})
}

func groupsOf(b *pob.PathOfBuilding) *[]pob.Skill {
	if len(b.Skills.SkillSets) == 0 {
		b.Skills.SkillSets = []pob.SkillSet{{ID: 1}}
		b.Skills.ActiveSkillSet = 1
	}
	return &b.Skills.SkillSets[b.ActiveSkillSetIndex()].Skills
}

func groupArg(b *pob.PathOfBuilding, group int) (*pob.Skill, *ToolError) {
	groups := *groupsOf(b)
	if group < 1 || group > len(groups) {
		e := errf(CodeInvalidValue, "No socket group %d.", group)
		if len(groups) > 0 {
			e.Allowed = []any{fmt.Sprintf("1-%d", len(groups))}
		}
		e.Hint = "See get_build section=skills."
		return nil, e
	}
	return &(*groupsOf(b))[group-1], nil
}

// checkSupports lists, once calculated, the supports in group that support none of its active skills; PoB allows them.
func checkSupports(ec *editCtx, group int, names []string) {
	ec.after = append(ec.after, func(env *calculator.Environment) {
		var list []gemEntry
		for _, n := range names {
			if g, e := resolveGem(n); e == nil && g.Support {
				list = append(list, g)
			}
		}
		for i, ok := range supportCompat(env, group, list) {
			if !ok {
				ec.add("unsupported", map[string]any{"group": group, "gem": list[i].Name})
			}
		}
	})
}

func buildGems(names []string, level, quality *int) ([]pob.Gem, []string, *ToolError) {
	var out []pob.Gem
	var resolved []string
	for _, n := range names {
		g, e := resolveGem(n)
		if e != nil {
			return nil, nil, e
		}
		lv, q := 20, 20
		if level != nil {
			lv = *level
		}
		if quality != nil {
			q = *quality
		}
		out = append(out, newGem(g, lv, q))
		resolved = append(resolved, g.Name)
	}
	return out, resolved, nil
}

func applySkillOp(ec *editCtx, op skillOp) *ToolError {
	b := ec.b
	switch op.Op {
	case "add_group":
		if len(op.Gems) == 0 {
			return invalidArg("add_group needs gems (active skill first).", nil)
		}
		list, names, e := buildGems(op.Gems, op.Level, op.Quality)
		if e != nil {
			return e
		}
		slot := ""
		if op.Slot != "" {
			if slot, e = resolveSlot(op.Slot); e != nil {
				return e
			}
		}
		groups := groupsOf(b)
		g := pob.Skill{Enabled: true, Slot: slot, Gems: list, IncludeInFullDPS: op.FullDPS}
		if len(*groups) == 1 && len((*groups)[0].Gems) == 0 {
			(*groups)[0] = g
		} else {
			*groups = append(*groups, g)
		}
		idx := len(*groups)
		if main := b.Build.MainSocketGroup; main < 1 || main > len(*groups) || len((*groups)[main-1].Gems) == 0 || main == idx {
			b.SetMainSocketGroup(idx)
		}
		ec.note(fmt.Sprintf("add_group %d: %s", idx, strings.Join(names, ", ")))
		checkSupports(ec, idx, names)
	case "set_gems":
		g, e := groupArg(b, op.Group)
		if e != nil {
			return e
		}
		list, names, e := buildGems(op.Gems, op.Level, op.Quality)
		if e != nil {
			return e
		}
		g.Gems = list
		if op.FullDPS != nil {
			g.IncludeInFullDPS = op.FullDPS
		}
		ec.note(fmt.Sprintf("set_gems %d: %s", op.Group, strings.Join(names, ", ")))
		checkSupports(ec, op.Group, names)
	case "add_gem":
		g, e := groupArg(b, op.Group)
		if e != nil {
			return e
		}
		list, names, e := buildGems([]string{op.Gem}, op.Level, op.Quality)
		if e != nil {
			return e
		}
		g.Gems = append(g.Gems, list...)
		ec.note(fmt.Sprintf("add_gem %s to group %d", names[0], op.Group))
		checkSupports(ec, op.Group, names)
	case "remove_gem":
		g, e := groupArg(b, op.Group)
		if e != nil {
			return e
		}
		i := slices.IndexFunc(g.Gems, func(x pob.Gem) bool { return gemMatches(x, op.Gem) })
		if i < 0 {
			return gemNotInGroup(g, op.Gem, op.Group)
		}
		ec.note(fmt.Sprintf("remove_gem %s from group %d", g.Gems[i].NameSpec, op.Group))
		g.Gems = slices.Delete(g.Gems, i, i+1)
	case "set_main":
		g, e := groupArg(b, op.Group)
		if e != nil {
			return e
		}
		b.SetMainSocketGroup(op.Group)
		if op.Gem != "" {
			k := 0
			found := false
			for _, gem := range g.Gems {
				if !gem.Enabled || gemIsSupport(gem) {
					continue
				}
				k++
				if gemMatches(gem, op.Gem) {
					g.MainActiveSkill = k
					found = true
					break
				}
			}
			if !found {
				return gemNotInGroup(g, op.Gem, op.Group)
			}
		}
		ec.note(fmt.Sprintf("set_main group %d", op.Group))
	case "enable_group", "disable_group":
		g, e := groupArg(b, op.Group)
		if e != nil {
			return e
		}
		g.Enabled = op.Op == "enable_group"
		ec.note(fmt.Sprintf("%s %d", op.Op, op.Group))
	case "remove_group":
		if _, e := groupArg(b, op.Group); e != nil {
			return e
		}
		b.DeleteSocketGroup(op.Group - 1)
		if main := b.Build.MainSocketGroup; main > op.Group {
			b.SetMainSocketGroup(main - 1)
		} else if main == op.Group {
			b.SetMainSocketGroup(b.GuessMainSocketGroup())
		}
		ec.note(fmt.Sprintf("remove_group %d", op.Group))
	case "set_gem_level":
		g, e := groupArg(b, op.Group)
		if e != nil {
			return e
		}
		i := slices.IndexFunc(g.Gems, func(x pob.Gem) bool { return gemMatches(x, op.Gem) })
		if i < 0 {
			return gemNotInGroup(g, op.Gem, op.Group)
		}
		if op.Level != nil {
			g.Gems[i].Level = *op.Level
		}
		if op.Quality != nil {
			g.Gems[i].Quality = *op.Quality
		}
		ec.note(fmt.Sprintf("set %s to level %d quality %d", g.Gems[i].NameSpec, g.Gems[i].Level, g.Gems[i].Quality))
	default:
		return invalidArg(fmt.Sprintf("Unknown op %q.", op.Op), skillOps)
	}
	return nil
}

func gemNotInGroup(g *pob.Skill, name string, group int) *ToolError {
	names := make([]string, 0, len(g.Gems))
	for _, gem := range g.Gems {
		names = append(names, gem.NameSpec)
	}
	e := errf(CodeNotFound, "Group %d has no gem %q.", group, name)
	e.DidYouMean = suggest(name, names, 5)
	e.Candidates = toAny(names)
	return e
}
