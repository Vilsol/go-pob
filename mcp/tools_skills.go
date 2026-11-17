package mcp

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strconv"
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
	if base := legacyGemBase(name); base != "" {
		return gemEntry{}, notFound("gem", name, []string{base}, fmt.Sprintf("%s is a legacy gem PoB does not offer for this game version; use %s.", name, base))
	}
	return gemEntry{}, notFound("gem", name, suggest(name, gemNames(nil), 5), "Retry with one of did_you_mean, or search_data kind=gem.")
}

// legacyGemBase names the gem a legacy gem called name is the plus version of, or "".
func legacyGemBase(name string) string {
	for _, rows := range []map[string]raw.PoBGem{raw.Gems, raw.GemVariants} {
		for _, row := range rows {
			skill := raw.PoBSkills[row.GrantedEffectID]
			if !skill.Legacy || (norm(row.Name) != norm(name) && norm(row.Name) != strings.TrimSuffix(norm(name), " support")) {
				continue
			}
			if base, ok := raw.PoBSkills[skill.PlusVersionOf]; ok && base.Name != "" {
				return base.Name
			}
		}
	}
	return ""
}

// gemRowByName finds the gem a name-only gem (no gem or skill id) resolves to, legacy gems included.
//
//pob:block:src/Classes/SkillsTab.lua:1203-1211
func gemRowByName(name string) (raw.PoBGem, bool) {
	for _, rows := range []map[string]raw.PoBGem{raw.Gems, raw.GemVariants} {
		for _, row := range rows {
			if norm(row.Name) == norm(name) || norm(row.Name) == strings.TrimSuffix(norm(name), " support") {
				return row, true
			}
		}
	}
	return raw.PoBGem{}, false
}

// legacyAwakenedNote says which awakened supports PoB still offers; the rest are legacy.
func legacyAwakenedNote() string {
	var offered []string
	for _, rows := range []map[string]raw.PoBGem{raw.Gems, raw.GemVariants} {
		for _, row := range rows {
			skill := raw.PoBSkills[row.GrantedEffectID]
			if skill.Support && skill.PlusVersionOf != "" && !skill.Legacy && strings.HasPrefix(row.Name, "Awakened ") {
				offered = append(offered, row.Name)
			}
		}
	}
	slices.Sort(offered)
	offered = slices.Compact(offered)
	return "Awakened supports are legacy here except " + strings.Join(offered, ", ") + "."
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
	Enabled *bool    `json:"enabled"`
	Part    any      `json:"part"`
}

type editSkillsIn struct {
	Handle string    `json:"handle"`
	DryRun bool      `json:"dry_run"`
	Watch  []string  `json:"watch"`
	Ops    []skillOp `json:"ops"`
}

var skillOps = []string{"add_group", "set_gems", "add_gem", "remove_gem", "set_main", "set_slot", "enable_group", "disable_group", "remove_group", "set_gem_level", "set_part"}

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

// checkSupports lists, once calculated, the supports in group (an index the batch's ops give) that support none of its
// active skills; PoB allows them.
func checkSupports(ec *editCtx, label int, names []string) {
	ec.after = append(ec.after, func(env *calculator.Environment) {
		group := ec.groupMap[label-1]
		if group == 0 {
			return
		}
		var list []gemEntry
		for _, n := range names {
			if g, e := resolveGem(n); e == nil && g.Support {
				list = append(list, g)
			}
		}
		for i, ok := range supportCompat(env, group, list) {
			if !ok {
				ec.add("unsupported", map[string]any{"group": label, "gem": list[i].Name})
			}
		}
	})
}

// buildGems resolves names into gems. A gem already in keep stays as it is. A new gem takes the place of a gem of keep
// that leaves, in order, as PoB's skills tab changes the gem in a slot; past those it is added to the group. Level and
// quality apply to the new gems when given.
func buildGems(b *pob.PathOfBuilding, names []string, level, quality *int, keep []pob.Gem) ([]pob.Gem, []string, *ToolError) {
	var resolved []gemEntry
	used := make([]bool, len(keep))
	stays := make([]int, len(names))
	for k, n := range names {
		g, e := resolveGem(n)
		if e != nil {
			return nil, nil, e
		}
		resolved = append(resolved, g)
		stays[k] = slices.IndexFunc(keep, func(x pob.Gem) bool { return gemMatches(x, g.Name) })
		if stays[k] >= 0 && used[stays[k]] {
			stays[k] = -1
		}
		if stays[k] >= 0 {
			used[stays[k]] = true
		}
	}
	var leaving []pob.Gem
	for i, x := range keep {
		if !used[i] {
			leaving = append(leaving, x)
		}
	}
	out := make([]pob.Gem, len(names))
	names = make([]string, len(resolved))
	for k, g := range resolved {
		names[k] = g.Name
		switch {
		case stays[k] >= 0:
			out[k] = keep[stays[k]]
		case len(leaving) > 0:
			out[k] = replacedGem(b, leaving[0], g, level, quality)
			leaving = leaving[1:]
		default:
			out[k] = addedGem(b, g, level, quality)
		}
	}
	return out, names, nil
}

// gemLevel is level when given, else the build's default gem level held to the highest level the character meets the
// level requirement of.
func gemLevel(b *pob.PathOfBuilding, g gemEntry, level *int) int {
	if level != nil {
		return *level
	}
	lv := calculator.ProcessGemLevel(b, g.GemID, false)
	if lv == 0 {
		lv = g.MaxLevel
	}
	return usableGemLevel(g.SkillID, lv, characterLevel(b))
}

// characterLevel is the build's character level, 1 when unset, as ProcessGemLevel reads it.
func characterLevel(b *pob.PathOfBuilding) int {
	return max(b.Build.Level, 1)
}

// gemLevelReq is the character level skillID's gem needs at level; 0 when the game data has no such level.
func gemLevelReq(skillID string, level int) int {
	ge := poe.GrantedEffectByID(skillID)
	if ge == nil {
		return 0
	}
	calc := raw.GetCalculatedGrantedEffect(ge)
	if calc == nil {
		return 0
	}
	if l, ok := calc.GetCalculatedLevels()[level]; ok {
		return l.LevelRequirement
	}
	return 0
}

// usableGemLevel is the highest gem level up to level whose requirement charLevel meets, at least 1.
//
//pob:block:src/Classes/SkillsTab.lua:1154-1163
func usableGemLevel(skillID string, level, charLevel int) int {
	for lv := level; lv > 1; lv-- {
		if gemLevelReq(skillID, lv) <= charLevel {
			return lv
		}
	}
	return 1
}

// addedGem is g as PoB's skills tab adds it to a group: the build's default level and quality, unless given.
//
//pob:block:src/Classes/SkillsTab.lua:796-799
//pob:block:src/Classes/SkillsTab.lua:827-829
func addedGem(b *pob.PathOfBuilding, g gemEntry, level, quality *int) pob.Gem {
	q := 0
	if b.Skills.DefaultGemQuality != nil {
		q = *b.Skills.DefaultGemQuality
	}
	if quality != nil {
		q = *quality
	}
	return newGem(g, gemLevel(b, g, level), q)
}

// replacedGem is g chosen in the skills tab slot that held old: the slot keeps its quality, count and enable states and
// the level is the new gem's default, unless given.
//
//pob:block:src/Classes/SkillsTab.lua:824-831
func replacedGem(b *pob.PathOfBuilding, old pob.Gem, g gemEntry, level, quality *int) pob.Gem {
	gem := keepSlot(newGem(g, gemLevel(b, g, level), 0), old)
	if quality != nil {
		gem.Quality = *quality
	}
	return gem
}

// gemLabels names gems as PoB's skill text does, "name level/quality".
//
//pob:block:src/Classes/SkillsTab.lua:704
func gemLabels(list []pob.Gem) string {
	out := make([]string, len(list))
	for i, g := range list {
		out[i] = fmt.Sprintf("%s %d/%d", g.NameSpec, g.Level, g.Quality)
		if !g.Enabled {
			out[i] += " (disabled)"
		}
	}
	return strings.Join(out, ", ")
}

// keepSlot is old with gem's identity and level: PoB's skills tab changes a slot's gemId and level in the same gem
// instance, so its quality, enable states, count and skill part, minion and stage choices stay.
//
//pob:block:src/Classes/SkillsTab.lua:824-829
func keepSlot(gem, old pob.Gem) pob.Gem {
	out := old
	out.GemID, out.NameSpec, out.SkillID, out.VariantID, out.Level = gem.GemID, gem.NameSpec, gem.SkillID, gem.VariantID, gem.Level
	return out
}

// initGroups records the socket groups as they are before the batch's first skill op.
func (ec *editCtx) initGroups() {
	if ec.groupMap != nil {
		return
	}
	groups := *groupsOf(ec.b)
	ec.groupMap = make([]int, len(groups))
	ec.origGroups = make([]pob.Skill, len(groups))
	for i, g := range groups {
		ec.groupMap[i] = i + 1
		g.Gems = slices.Clone(g.Gems)
		ec.origGroups[i] = g
	}
}

// groupIndex resolves a group index an op gives, which names the groups as they were before the batch (groups
// add_group creates are numbered after them), to the group's index now.
func (ec *editCtx) groupIndex(label int) (int, *ToolError) {
	ec.initGroups()
	if label < 1 || label > len(ec.groupMap) {
		e := errf(CodeInvalidValue, "No socket group %d.", label)
		if len(ec.groupMap) > 0 {
			e.Allowed = []any{fmt.Sprintf("1-%d", len(ec.groupMap))}
		}
		e.Hint = "See get_build section=skills; indexes in a batch name the groups as they were before it."
		return 0, e
	}
	if ec.groupMap[label-1] == 0 {
		e := errf(CodeInvalidValue, "Socket group %d was removed earlier in this batch.", label)
		e.Hint = "Indexes in a batch name the groups as they were before it."
		return 0, e
	}
	return ec.groupMap[label-1], nil
}

// gemGroupLabel is the group index (as the batch names groups) of the one gem named name in the active skill set; a
// name no group or several hold is an error naming the groups.
func (ec *editCtx) gemGroupLabel(name string) (int, *ToolError) {
	var labels []any
	for i, cur := range ec.groupMap {
		if cur == 0 {
			continue
		}
		for _, gem := range (*groupsOf(ec.b))[cur-1].Gems {
			if gemMatches(gem, name) {
				labels = append(labels, i+1)
			}
		}
	}
	switch len(labels) {
	case 1:
		return labels[0].(int), nil
	case 0:
		return 0, errf(CodeNotFound, "No socket group has a gem %q.", name)
	}
	e := errf(CodeAmbiguous, "Several gems %q: name the group.", name)
	e.Candidates = labels
	return 0, e
}

// noteGroupIndexes returns, once a group has moved, each moved group's index before the batch and after it.
func (ec *editCtx) noteGroupIndexes() {
	moved := map[string]int{}
	for i, cur := range ec.groupMap {
		if cur != 0 && cur != i+1 {
			moved[strconv.Itoa(i+1)] = cur
		}
	}
	if ec.extra == nil {
		ec.extra = map[string]any{}
	}
	if len(moved) > 0 {
		ec.extra["group_indexes"] = moved
	} else {
		delete(ec.extra, "group_indexes")
	}
}

// groupGemCount is the count Full DPS takes for a group without its own count: its first enabled active gem's.
//
//pob:block:src/Modules/Calcs.lua:126-142
func groupGemCount(gems []pob.Gem) int {
	for _, gem := range gems {
		if gem.Enabled && !gemIsSupport(gem) && gem.Count != nil {
			return *gem.Count
		}
	}
	return 1
}

func applySkillOp(ec *editCtx, op skillOp) *ToolError {
	if !slices.Contains(skillOps, op.Op) {
		return invalidArg(fmt.Sprintf("Unknown op %q.", op.Op), skillOps)
	}
	ec.initGroups()
	if op.Op == "set_gem_level" && op.Group == 0 && op.Gem != "" {
		label, e := ec.gemGroupLabel(op.Gem)
		if e != nil {
			return e
		}
		op.Group = label
	}
	label := op.Group
	if op.Op != "add_group" {
		cur, e := ec.groupIndex(label)
		if e != nil {
			return e
		}
		op.Group = cur
	}
	e := applyGroupOp(ec, op, label)
	if e == nil && op.Op == "remove_group" {
		for i, c := range ec.groupMap {
			switch {
			case c == op.Group:
				ec.groupMap[i] = 0
			case c > op.Group:
				ec.groupMap[i] = c - 1
			}
		}
	}
	ec.noteGroupIndexes()
	return e
}

// applyGroupOp applies op, whose group is the index now; label is the index the op gave.
func applyGroupOp(ec *editCtx, op skillOp, label int) *ToolError {
	b := ec.b
	switch op.Op {
	case "add_group":
		if len(op.Gems) == 0 {
			return invalidArg("add_group needs gems (active skill first).", nil)
		}
		list, names, e := buildGems(b, op.Gems, op.Level, op.Quality, nil)
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
		if idx > 1 || len(ec.groupMap) == 0 {
			ec.groupMap = append(ec.groupMap, idx)
		}
		label = len(ec.groupMap)
		if main := b.Build.MainSocketGroup; main < 1 || main > len(*groups) || len((*groups)[main-1].Gems) == 0 || main == idx {
			b.SetMainSocketGroup(idx)
		}
		if slot != "" {
			ec.note(fmt.Sprintf("add_group %d in %s: %s", label, slot, gemLabels(list)))
		} else {
			ec.note(fmt.Sprintf("add_group %d (no slot): %s", label, gemLabels(list)))
		}
		checkSupports(ec, label, names)
	case "set_gems":
		g, e := groupArg(b, op.Group)
		if e != nil {
			return e
		}
		list, names, e := buildGems(b, op.Gems, op.Level, op.Quality, g.Gems)
		if e != nil {
			return e
		}
		// PoB's skills tab keeps a slot's enabled state when its gem changes (SkillsTab.lua:823-830)
		var enabled, disabled, turnedOff []string
		for i := range list {
			if op.Enabled != nil && !*op.Enabled {
				if list[i].Enabled {
					list[i].Enabled = false
					turnedOff = append(turnedOff, list[i].NameSpec)
				}
				continue
			}
			if list[i].Enabled {
				continue
			}
			if op.Enabled != nil && *op.Enabled {
				list[i].Enabled = true
				enabled = append(enabled, list[i].NameSpec)
			} else {
				disabled = append(disabled, list[i].NameSpec)
			}
		}
		g.Gems = list
		if op.FullDPS != nil {
			g.IncludeInFullDPS = op.FullDPS
		}
		if op.Slot != "" {
			if g.Slot, e = resolveSlot(op.Slot); e != nil {
				return e
			}
		}
		ec.note(fmt.Sprintf("set_gems %d: %s", label, gemLabels(list)))
		if len(enabled) > 0 {
			ec.note(fmt.Sprintf("set_gems %d enabled %s, which were disabled", label, strings.Join(enabled, ", ")))
		}
		if len(turnedOff) > 0 {
			ec.note(fmt.Sprintf("set_gems %d disabled %s", label, strings.Join(turnedOff, ", ")))
		}
		if len(disabled) > 0 {
			ec.note(fmt.Sprintf("set_gems %d kept %s disabled, as Path of Building keeps a slot's enabled state; pass enabled:true to enable them", label, strings.Join(disabled, ", ")))
		}
		checkSupports(ec, label, names)
	case "add_gem":
		g, e := groupArg(b, op.Group)
		if e != nil {
			return e
		}
		list, names, e := buildGems(b, []string{op.Gem}, op.Level, op.Quality, nil)
		if e != nil {
			return e
		}
		if !gemIsSupport(list[0]) && g.GroupCount == nil {
			count := groupGemCount(g.Gems)
			if count == 1 && label <= len(ec.origGroups) {
				count = groupGemCount(ec.origGroups[label-1].Gems)
			}
			if count > 1 {
				list[0].Count = &count
			}
		}
		g.Gems = append(g.Gems, list...)
		if c := list[0].Count; c != nil && *c > 1 {
			ec.note(fmt.Sprintf("add_gem %s to group %d, counted %d times in Full DPS as the group's active gem was", gemLabels(list), label, *c))
		} else {
			ec.note(fmt.Sprintf("add_gem %s to group %d", gemLabels(list), label))
		}
		checkSupports(ec, label, names)
	case "remove_gem":
		g, e := groupArg(b, op.Group)
		if e != nil {
			return e
		}
		i := slices.IndexFunc(g.Gems, func(x pob.Gem) bool { return gemMatches(x, op.Gem) })
		if i < 0 {
			return gemNotInGroup(g, op.Gem, label)
		}
		ec.note(fmt.Sprintf("remove_gem %s from group %d", g.Gems[i].NameSpec, label))
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
				return gemNotInGroup(g, op.Gem, label)
			}
		}
		ec.note(fmt.Sprintf("set_main group %d", label))
	case "set_slot":
		g, e := groupArg(b, op.Group)
		if e != nil {
			return e
		}
		if op.Slot == "" {
			return invalidArg("set_slot needs slot.", slotNames())
		}
		if g.Slot, e = resolveSlot(op.Slot); e != nil {
			return e
		}
		ec.note(fmt.Sprintf("set_slot %d: %s", label, g.Slot))
	case "enable_group", "disable_group":
		g, e := groupArg(b, op.Group)
		if e != nil {
			return e
		}
		g.Enabled = op.Op == "enable_group"
		ec.note(fmt.Sprintf("%s %d", op.Op, label))
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
		ec.note(fmt.Sprintf("remove_group %d", label))
	case "set_gem_level":
		g, e := groupArg(b, op.Group)
		if e != nil {
			return e
		}
		i := slices.IndexFunc(g.Gems, func(x pob.Gem) bool { return gemMatches(x, op.Gem) })
		if i < 0 {
			return gemNotInGroup(g, op.Gem, label)
		}
		if op.Level != nil {
			g.Gems[i].Level = *op.Level
		}
		if op.Quality != nil {
			g.Gems[i].Quality = *op.Quality
		}
		ec.note(fmt.Sprintf("set %s to level %d quality %d", g.Gems[i].NameSpec, g.Gems[i].Level, g.Gems[i].Quality))
	case "set_part":
		g, e := groupArg(b, op.Group)
		if e != nil {
			return e
		}
		i := mainGemIndex(g)
		if op.Gem != "" {
			i = slices.IndexFunc(g.Gems, func(x pob.Gem) bool { return gemMatches(x, op.Gem) })
			if i < 0 {
				return gemNotInGroup(g, op.Gem, label)
			}
		}
		if i < 0 {
			return invalidArg(fmt.Sprintf("Group %d has no active gem; name one with gem.", label), nil)
		}
		parts := gemParts(g.Gems[i])
		if len(parts) < 2 {
			return invalidArg(fmt.Sprintf("%s has no skill parts to choose from.", g.Gems[i].NameSpec), nil)
		}
		k, e := partIndex(parts, op.Part)
		if e != nil {
			return e
		}
		g.Gems[i].SkillPart = k
		ec.note(fmt.Sprintf("set_part group %d: %s part %d %s", label, g.Gems[i].NameSpec, k, parts[k-1]))
	default:
		return invalidArg(fmt.Sprintf("Unknown op %q.", op.Op), skillOps)
	}
	return nil
}

// gemParts names the parts of a gem's skill, the list PoB's main skill part control offers (Build.lua:1658-1664).
func gemParts(gem pob.Gem) []string {
	parts := raw.SkillParts[gemSkillID(gem)]
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, p.Name)
	}
	return out
}

// gemPart is the part the calculation uses: the gem's skillPart, clamped to the parts (CalcActiveSkill.lua:194-200).
func gemPart(gem pob.Gem, parts []string) int {
	return min(max(gem.SkillPart, 1), len(parts))
}

// mainGemIndex is the index of the group's main active gem, mainActiveSkill counting enabled active gems, or -1.
func mainGemIndex(g *pob.Skill) int {
	k := 0
	for i, gem := range g.Gems {
		if !gem.Enabled || gemIsSupport(gem) {
			continue
		}
		k++
		if k == max(g.MainActiveSkill, 1) {
			return i
		}
	}
	return -1
}

// partIndex resolves a part given as its 1-based index or its name.
func partIndex(parts []string, v any) (int, *ToolError) {
	switch p := v.(type) {
	case float64:
		if k := int(p); float64(k) == p && k >= 1 && k <= len(parts) {
			return k, nil
		}
	case string:
		if k, err := strconv.Atoi(p); err == nil && k >= 1 && k <= len(parts) {
			return k, nil
		}
		if k := slices.IndexFunc(parts, func(n string) bool { return norm(n) == norm(p) }); k >= 0 {
			return k + 1, nil
		}
	case nil:
		return 0, invalidArg("set_part needs part: a part name or its 1-based index.", parts)
	}
	return 0, invalidArg(fmt.Sprintf("Unknown part %v.", v), parts)
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

// maxLinks is the size of an item's largest linked socket group.
func maxLinks(item *calculator.Item) int {
	links := 0
	for i, run := 0, 0; i < len(item.Sockets); i++ {
		if i == 0 || item.Sockets[i].Group != item.Sockets[i-1].Group {
			run = 0
		}
		run++
		links = max(links, run)
	}
	return links
}

const unsocketedNote = "not socketed: gets no links in game (PoB still applies its supports); set a slot with edit_skills op=set_slot"

// unsocketed describes each enabled group of gems that sits in no item, with the most-linked equipped item no
// group uses yet.
func unsocketed(b *pob.PathOfBuilding) map[int]map[string]any {
	groups := activeSkills(b)
	var loose []int
	taken := map[string]bool{}
	for i, g := range groups {
		taken[g.Slot] = true
		if g.Enabled && g.Slot == "" && g.Source == "" && g.SourceAttr == nil && slices.ContainsFunc(g.Gems, func(x pob.Gem) bool { return x.Enabled }) {
			loose = append(loose, i)
		}
	}
	out := map[int]map[string]any{}
	if len(loose) == 0 {
		return out
	}
	bestSlot, bestLinks := "", 0
	for _, e := range equippedItems(b) {
		if taken[e.slot] || strings.HasPrefix(e.slot, "Flask") {
			continue
		}
		if l := maxLinks(calculator.ParseItem(e.text)); l > bestLinks {
			bestSlot, bestLinks = e.slot, l
		}
	}
	for _, i := range loose {
		u := map[string]any{"group": i + 1, "note": unsocketedNote}
		if bestSlot != "" {
			u["best_slot"] = bestSlot
			u["links"] = bestLinks
		} else {
			u["note"] = unsocketedNote + " after equipping an item with linked sockets"
		}
		out[i+1] = u
	}
	return out
}

// unsocketedList is unsocketed in group order, at most three groups.
func unsocketedList(b *pob.PathOfBuilding) []any {
	byGroup := unsocketed(b)
	var out []any
	for i := 1; i <= len(activeSkills(b)) && len(out) < 3; i++ {
		if u, ok := byGroup[i]; ok {
			out = append(out, u)
		}
	}
	return out
}
