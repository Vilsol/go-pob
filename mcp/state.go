package mcp

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
)

const (
	maxHandles     = 128
	maxWarmHandles = 8
	maxSnapshots   = 50
)

type revision struct {
	xml []byte
	env *calculator.Environment
}

type handle struct {
	id       string
	name     string
	revs     []*revision
	cur      int
	exported bool
	versions map[string]any
	lastUsed int64
	// source names where the build came from, again the call that recreates it (empty when that needs the original
	// code or XML) and saved the last file export_build wrote of it
	source   string
	again    string
	saved    string
	savedRev int
	// parents holds the revision an edit made after a revert built on; every other revision's is the one before it
	parents map[int]int
}

// parent is the revision rev was edited from, -1 for revision 0.
func (h *handle) parent(rev int) int {
	if p, ok := h.parents[rev]; ok {
		return p
	}
	return rev - 1
}

type evictedHandle struct {
	why    string
	source string
	hint   string
}

// reimport says how to get an evicted handle's build back in one call.
func (h *handle) reimport() string {
	switch {
	case h.saved != "":
		return fmt.Sprintf("Re-import its last export (revision %d): import_build source=%s.", h.savedRev, h.saved)
	case h.again != "":
		return "Re-create it with " + h.again + "; edits made on the handle are not in it."
	}
	return "Re-import the build with import_build from " + h.source + "."
}

func (h *handle) latest() int {
	return len(h.revs) - 1
}

func (s *Server) handleIDs() []string {
	return append([]string(nil), s.order...)
}

func (s *Server) get(id string) (*handle, *ToolError) {
	h, ok := s.handles[id]
	if !ok {
		e := errf(CodeUnknownHandle, "No live build handle %q.", id)
		if ev, gone := s.evicted[id]; gone {
			e.Message = fmt.Sprintf("Build handle %q was evicted: %s It came from %s.", id, ev.why, ev.source)
			e.Hint = ev.hint + " Continue on the new handle."
		}
		e.ValidHandles = s.handleIDs()
		if len(e.ValidHandles) == 0 && e.Hint == "" {
			e.Hint = "Create one with new_build or import_build."
		}
		return nil, e
	}
	s.touch(h)
	return h, nil
}

// touch marks h as the most recently used handle and drops the cached calculations of handles outside the
// maxWarmHandles most recently used; their snapshots stay and recalculate on next use.
func (s *Server) touch(h *handle) {
	s.useSeq++
	h.lastUsed = s.useSeq
	if len(s.order) <= maxWarmHandles {
		return
	}
	byUse := s.handlesByUse()
	for _, c := range byUse[:len(byUse)-maxWarmHandles] {
		for _, r := range c.revs {
			if r != nil {
				r.env = nil
			}
		}
	}
}

// handlesByUse lists live handles from least to most recently used.
func (s *Server) handlesByUse() []*handle {
	out := make([]*handle, 0, len(s.order))
	for _, id := range s.order {
		out = append(out, s.handles[id])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].lastUsed < out[j].lastUsed })
	return out
}

func (h *handle) check(rev int) *ToolError {
	if rev < 0 || rev >= len(h.revs) {
		e := errf(CodeInvalidValue, "Handle %s has no revision %d.", h.id, rev)
		e.Allowed = []any{fmt.Sprintf("0-%d", h.latest())}
		return e
	}
	if h.revs[rev] == nil {
		e := errf(CodeInvalidValue, "Revision %d of %s is older than the %d kept snapshots.", rev, h.id, maxSnapshots)
		e.Allowed = []any{fmt.Sprintf("%d-%d", h.latest()-maxSnapshots+1, h.latest())}
		return e
	}
	return nil
}

// parseRev re-parses a snapshot, so every edit and every calculation starts from its own copy of the build.
func (h *handle) parseRev(rev int) (*pob.PathOfBuilding, *ToolError) {
	if e := h.check(rev); e != nil {
		return nil, e
	}
	b, err := builds.ParseBuild(h.revs[rev].xml)
	if err != nil {
		return nil, errf(CodeInternal, "snapshot does not parse: %v", err)
	}
	return b, nil
}

func calcEnv(b *pob.PathOfBuilding) *calculator.Environment {
	return calculator.NewCalculator(*b).BuildOutput(calculator.OutputModeMain)
}

func (h *handle) envAt(rev int) (*calculator.Environment, *ToolError) {
	if e := h.check(rev); e != nil {
		return nil, e
	}
	r := h.revs[rev]
	if r.env == nil {
		b, e := h.parseRev(rev)
		if e != nil {
			return nil, e
		}
		r.env = calcEnv(b)
	}
	return r.env, nil
}

func serialize(b *pob.PathOfBuilding) ([]byte, *ToolError) {
	out, err := builds.SerializeBuild(b)
	if err != nil {
		return nil, errf(CodeInternal, "the build does not serialize: %v", err)
	}
	return out, nil
}

// commit appends b as the handle's next revision and makes it current.
func (s *Server) commit(h *handle, b *pob.PathOfBuilding) (int, *ToolError) {
	xml, env, e := calculate(b)
	if e != nil {
		return 0, e
	}
	return s.commitCalculated(h, xml, env)
}

// calculate serializes b and calculates the snapshot, applying the level estimate a build in characterLevelAutoMode
// follows (Build.lua:910-914: PoB sets the level the allocated points need and recalculates).
func calculate(b *pob.PathOfBuilding) ([]byte, *calculator.Environment, *ToolError) {
	xml, e := serialize(b)
	if e != nil {
		return nil, nil, e
	}
	h := &handle{revs: []*revision{{xml: xml}}}
	env, e := h.envAt(0)
	if e != nil || !b.Build.CharacterLevelAutoMode {
		return xml, env, e
	}
	level := pointsOf(env).LevelNeeded
	if level == b.Build.Level {
		return xml, env, nil
	}
	b.Build.Level = level
	if xml, e = serialize(b); e != nil {
		return nil, nil, e
	}
	h = &handle{revs: []*revision{{xml: xml}}}
	env, e = h.envAt(0)
	return xml, env, e
}

// commitCalculated appends a serialized build, with its environment when the caller already calculated it.
func (s *Server) commitCalculated(h *handle, xml []byte, env *calculator.Environment) (int, *ToolError) {
	if h.cur != h.latest() {
		if h.parents == nil {
			h.parents = map[int]int{}
		}
		h.parents[len(h.revs)] = h.cur
	}
	h.revs = append(h.revs, &revision{xml: xml, env: env})
	h.cur = h.latest()
	for i := range h.revs {
		if i < h.latest()-maxSnapshots+1 && i != 0 {
			h.revs[i] = nil
		} else if h.revs[i] != nil && i != h.cur && i != 0 && i < h.cur-2 {
			h.revs[i].env = nil
		}
	}
	if _, e := h.envAt(h.cur); e != nil {
		return 0, e
	}
	return h.cur, nil
}

func (s *Server) addHandle(b *pob.PathOfBuilding, name, source, again string, versions map[string]any) (*handle, *ToolError) {
	xml, env, e := calculate(b)
	if e != nil {
		return nil, e
	}
	s.nextHandle++
	id := "b" + strconv.Itoa(s.nextHandle)
	if len(s.order) >= maxHandles {
		byUse := s.handlesByUse()
		evict := byUse[0]
		for _, c := range byUse {
			if !c.exported {
				evict = c
				break
			}
		}
		s.evictHandle(evict.id)
		s.evicted[evict.id] = evictedHandle{fmt.Sprintf("it was the least recently used of %d live handles (the per-process limit) when %s was created.", maxHandles, id), evict.source, evict.reimport()}
		s.pendingWarnings = append(s.pendingWarnings, fmt.Sprintf("Handle %s was evicted: least recently used of %d live handles.", evict.id, maxHandles))
	}
	h := &handle{id: id, name: name, revs: []*revision{{xml: xml, env: env}}, versions: versions, source: source, again: again}
	s.handles[h.id] = h
	s.order = append(s.order, h.id)
	s.touch(h)
	if _, e := h.envAt(0); e != nil {
		return nil, e
	}
	return h, nil
}

func (s *Server) evictHandle(id string) {
	delete(s.handles, id)
	for i, o := range s.order {
		if o == id {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
}

func activeSpec(b *pob.PathOfBuilding) *pob.Spec {
	i := b.ActiveSpecIndex()
	if i < 0 || i >= len(b.Tree.Specs) {
		return nil
	}
	return &b.Tree.Specs[i]
}

func specTreeVersion(b *pob.PathOfBuilding) pob.TreeVersion {
	if s := activeSpec(b); s != nil && s.TreeVersion != "" {
		return s.TreeVersion
	}
	return data.LatestTreeVersion
}

func versionsOf(b *pob.PathOfBuilding, buildTree pob.TreeVersion, converted bool) map[string]any {
	v := map[string]any{"game_data": raw.LatestVersion, "tree": string(specTreeVersion(b))}
	if buildTree != "" {
		v["build_tree"] = string(buildTree)
		v["converted"] = converted
	}
	return v
}

type pointsInfo struct {
	Used          int `json:"used"`
	Available     int `json:"available"`
	AscUsed       int `json:"ascendancy_used"`
	LevelNeeded   int `json:"level_needed"`
	extraWarnings []string
}

func pointsOf(env *calculator.Environment) pointsInfo {
	used, asc, secondary, _ := env.CountAllocNodes()
	extra := int(env.Player.Output["ExtraPoints"])
	p := estimatePlayerProgress(used, asc, secondary, extra)
	level := env.Build.Build.Level
	return pointsInfo{Used: used, Available: max(level-1, 0) + 23 + extra, AscUsed: asc, LevelNeeded: p.Level, extraWarnings: p.Warnings}
}

// warningsOf assembles PoB's sidebar warnings (Build.lua:907-909, 1778-1831), the attribute/pool lines of
// BuildDisplayStats.lua that warn, then validationWarnings and unparsedWarnings.
func warningsOf(env *calculator.Environment) []string {
	var out []string
	push := func(t string) {
		for _, o := range out {
			if o == t {
				return
			}
		}
		out = append(out, t)
	}
	o := env.Player.Output
	for _, a := range []struct{ name, stat, req string }{{"Strength", "Str", "ReqStr"}, {"Dexterity", "Dex", "ReqDex"}, {"Intelligence", "Int", "ReqInt"}} {
		if o[a.stat] < o[a.req] {
			push(fmt.Sprintf("You do not meet the %s requirement", a.name))
		}
	}
	if v, ok := o["LifeUnreserved"]; ok && v <= 0 && o["Life"] > 0 {
		push("Your unreserved Life is below 1")
	}
	if v, ok := o["ManaUnreserved"]; ok && v < 0 {
		push("Your unreserved Mana is negative")
	}
	w := env.SidebarWarnings()
	if w != nil {
		for _, p := range []struct {
			pool string
			list []string
		}{{"Life", w.LifeCostWarningList}, {"Mana", w.ManaCostWarningList}, {"Rage", w.RageCostWarningList}, {"Energy Shield", w.ESCostWarningList}} {
			if len(p.list) > 0 {
				name := p.pool
				if w.EnergyShieldProtectsMana && p.pool == "Mana" {
					name = "Energy Shield and Mana"
				}
				push(fmt.Sprintf("You do not have enough %s to use: %s", name, strings.Join(p.list, ", ")))
			}
		}
		for _, p := range []struct {
			pool string
			list []string
		}{{"Unreserved life", w.LifePercentCostPercentCostWarningList}, {"Unreserved Mana", w.ManaPercentCostPercentCostWarningList}} {
			if len(p.list) > 0 {
				push(fmt.Sprintf("You do not have enough %s%% to use: %s", p.pool, strings.Join(p.list, ", ")))
			}
		}
		if w.PlayerAspectCount > 1 {
			push("You have more than one Aspect skill active")
		}
		for _, j := range w.JewelLimitWarning {
			push("You are exceeding jewel limit with the jewel " + j)
		}
		for _, sl := range w.SocketLimitWarning {
			push("You have too many gems in your " + sl + " slot")
		}
		if len(w.MissingAnointWarning) > 0 {
			push("You have eligible items missing an anoint: " + strings.Join(w.MissingAnointWarning, ", "))
		}
	}
	for _, l := range pointsOf(env).extraWarnings {
		push(l)
	}
	for _, l := range validationWarnings(env) {
		push(l)
	}
	for _, l := range gemWarnings(env.Build) {
		push(l)
	}
	for _, l := range totemWarnings(env) {
		push(l)
	}
	for _, l := range unparsedWarnings(env) {
		push(l)
	}
	if l := customModDefences(env); l != "" {
		push(l)
	}
	return out
}

type mainSkillInfo struct {
	Group    int      `json:"group"`
	Name     string   `json:"name"`
	Slot     string   `json:"slot,omitempty"`
	Supports []string `json:"supports"`
}

func activeSkills(b *pob.PathOfBuilding) []pob.Skill {
	if len(b.Skills.SkillSets) == 0 {
		return nil
	}
	return b.Skills.SkillSets[b.ActiveSkillSetIndex()].Skills
}

func gemIsSupport(g pob.Gem) bool {
	if g.VariantID != nil {
		if row, ok := raw.GemVariants[*g.VariantID]; ok {
			return row.Tags["support"]
		}
	}
	if row, ok := raw.Gems[g.GemID]; ok {
		return row.Tags["support"]
	}
	if g.GemID == "" && g.SkillID == "" {
		if row, ok := gemRowByName(g.NameSpec); ok {
			return row.Tags["support"]
		}
	}
	return strings.HasPrefix(g.SkillID, "Support")
}

func mainSkillOf(b *pob.PathOfBuilding) *mainSkillInfo {
	groups := activeSkills(b)
	i := b.Build.MainSocketGroup
	if i < 1 || i > len(groups) {
		return nil
	}
	g := groups[i-1]
	if len(g.Gems) == 0 {
		return nil
	}
	actives, supports := []string{}, []string{}
	for _, gem := range g.Gems {
		if !gem.Enabled {
			continue
		}
		if gemIsSupport(gem) {
			supports = append(supports, gem.NameSpec)
		} else {
			actives = append(actives, gem.NameSpec)
		}
	}
	if len(actives) == 0 {
		return &mainSkillInfo{Group: i, Name: g.Label, Slot: g.Slot, Supports: supports}
	}
	pick := min(max(g.MainActiveSkill, 1), len(actives))
	return &mainSkillInfo{Group: i, Name: actives[pick-1], Slot: g.Slot, Supports: supports}
}

func summaryOf(h *handle, rev int) (map[string]any, *ToolError) {
	b, e := h.parseRev(rev)
	if e != nil {
		return nil, e
	}
	env, e := h.envAt(rev)
	if e != nil {
		return nil, e
	}
	stats := map[string]float64{}
	for _, k := range summaryKeys {
		v, _ := statValue(env, k)
		if v != 0 || strings.HasSuffix(k, "Resist") || k == "Life" {
			stats[k] = round(v)
		}
	}
	out := map[string]any{
		"class":      string(b.Build.ClassName),
		"ascendancy": ascendancyName(b),
		"level":      b.Build.Level,
		"main_skill": mainSkillOf(b),
		"stats":      stats,
		"points":     pointsOf(env),
		"warnings":   nonNil(cachedWarnings(env)),
	}
	return out, nil
}

func ascendancyName(b *pob.PathOfBuilding) any {
	if b.Build.AscendClassName == "" || b.Build.AscendClassName == "None" {
		return nil
	}
	return string(b.Build.AscendClassName)
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
