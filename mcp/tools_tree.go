package mcp

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

type treeNode struct {
	ID         int64
	Name       string
	Type       string
	Stats      []string
	Ascendancy string
	Node       data.Node
}

var treeIndexes = map[pob.TreeVersion][]treeNode{}

func nodeType(n data.Node) string {
	switch {
	case n.AscendancyName != nil && *n.AscendancyName != "":
		return "ascendancy"
	case n.IsKeystone != nil && *n.IsKeystone:
		return "keystone"
	case n.IsMastery != nil && *n.IsMastery:
		return "mastery"
	case n.IsJewelSocket != nil && *n.IsJewelSocket:
		return "jewel"
	case n.IsNotable != nil && *n.IsNotable:
		return "notable"
	}
	return "small"
}

// treeIndex lists the tree's allocatable nodes: no class or ascendancy starts, proxies or cluster nodes.
func treeIndex(v pob.TreeVersion) ([]treeNode, *ToolError) {
	if list, ok := treeIndexes[v]; ok {
		return list, nil
	}
	tv, ok := data.TreeVersions[v]
	if !ok {
		return nil, errf(CodeInternal, "no passive tree data for version %s", v)
	}
	var list []treeNode
	for _, n := range tv.Tree().Nodes {
		if n.Skill == nil || n.Name == nil || *n.Skill >= 65536 || n.ClassStartIndex != nil ||
			(n.IsAscendancyStart != nil && *n.IsAscendancyStart) || (n.IsProxy != nil && *n.IsProxy) ||
			(n.IsBlighted != nil && *n.IsBlighted) || len(n.In)+len(n.Out) == 0 {
			continue
		}
		t := treeNode{ID: *n.Skill, Name: *n.Name, Type: nodeType(n), Stats: n.Stats, Node: n}
		if n.AscendancyName != nil {
			t.Ascendancy = *n.AscendancyName
		}
		list = append(list, t)
	}
	slices.SortFunc(list, func(a, b treeNode) int { return int(a.ID - b.ID) })
	treeIndexes[v] = list
	return list, nil
}

func treeNodeByID(v pob.TreeVersion, id int64) (treeNode, bool) {
	list, e := treeIndex(v)
	if e != nil {
		return treeNode{}, false
	}
	i, ok := slices.BinarySearchFunc(list, id, func(t treeNode, id int64) int { return int(t.ID - id) })
	if !ok {
		return treeNode{}, false
	}
	return list[i], true
}

// nodeArg reads a node reference: an integer id, a numeric string, or a name.
func nodeArg(raw json.RawMessage) (int64, string, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, "", false
	}
	var id int64
	if json.Unmarshal(raw, &id) == nil {
		return id, "", true
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		s = strings.TrimSpace(s)
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n, "", true
		}
		return 0, s, s != ""
	}
	return 0, "", false
}

// startNodes are the class start and ascendancy start the build holds without spending points.
func startNodes(b *pob.PathOfBuilding, v pob.TreeVersion) []int64 {
	tv := data.TreeVersions[v]
	if tv == nil {
		return nil
	}
	classID, ok := pob.ClassIDByName(b.Build.ClassName)
	if !ok {
		return nil
	}
	var out []int64
	for _, n := range tv.Tree().Nodes {
		if n.Skill == nil {
			continue
		}
		if n.ClassStartIndex != nil && *n.ClassStartIndex == int64(classID) {
			out = append(out, *n.Skill)
		}
		if n.IsAscendancyStart != nil && *n.IsAscendancyStart && n.AscendancyName != nil && *n.AscendancyName == string(b.Build.AscendClassName) {
			out = append(out, *n.Skill)
		}
	}
	return out
}

// classRoots are the main-tree neighbours of the class start (frontend skill_tree/index.ts classStartNodes).
func classRoots(b *pob.PathOfBuilding, v pob.TreeVersion) []int64 {
	tv := data.TreeVersions[v]
	if tv == nil {
		return nil
	}
	classID, _ := pob.ClassIDByName(b.Build.ClassName)
	nodes := tv.Tree().Nodes
	var out []int64
	for _, n := range nodes {
		if n.ClassStartIndex == nil || *n.ClassStartIndex != int64(classID) {
			continue
		}
		for _, o := range slices.Concat(n.Out, n.In) {
			t, ok := nodes[o]
			if ok && t.Skill != nil && (t.AscendancyName == nil || *t.AscendancyName == "") {
				out = append(out, *t.Skill)
			}
		}
	}
	return out
}

type pathPlan struct {
	hops   map[int64]int64
	active map[int64]bool
}

func planPaths(b *pob.PathOfBuilding, v pob.TreeVersion) pathPlan {
	active := map[int64]bool{}
	for _, id := range b.Build.PassiveNodes {
		active[id] = true
	}
	for _, id := range startNodes(b, v) {
		active[id] = true
	}
	list := make([]int64, 0, len(active))
	for id := range active {
		list = append(list, id)
	}
	slices.Sort(list)
	return pathPlan{hops: data.TreeVersions[v].CalculateAllocationPaths(list, nil), active: active}
}

// path walks the predecessor map from target back to the allocated tree; nil when unreachable.
func (p pathPlan) path(target int64) []int64 {
	if p.active[target] {
		return []int64{}
	}
	var out []int64
	for cur := target; !p.active[cur]; {
		next, ok := p.hops[cur]
		if !ok || next == -1 || len(out) > 200 {
			return nil
		}
		out = append(out, cur)
		cur = next
	}
	return out
}

func nodeNames(list []treeNode) []string {
	out := make([]string, len(list))
	for i, n := range list {
		out[i] = n.Name
	}
	return out
}

// resolveNode finds a node by id or name; same-named nodes resolve to the one with the cheapest path.
func resolveNode(b *pob.PathOfBuilding, v pob.TreeVersion, raw json.RawMessage, plan *pathPlan) (treeNode, *ToolError) {
	id, name, ok := nodeArg(raw)
	if !ok {
		return treeNode{}, invalidArg("node is required (an id or a name).", nil)
	}
	list, e := treeIndex(v)
	if e != nil {
		return treeNode{}, e
	}
	if name == "" {
		if n, ok := treeNodeByID(v, id); ok {
			return n, nil
		}
		return treeNode{}, notFound("passive node with id", strconv.FormatInt(id, 10), nil, "Find ids with search_data kind=node.")
	}
	_, ok, hits := resolveName(name, list, func(t treeNode) string { return t.Name }, func(t treeNode) string { return strings.Join(t.Stats, " ") })
	if !ok {
		if len(hits) > 0 {
			e := errf(CodeAmbiguous, "%q matches several nodes.", name)
			for _, h := range hits[:min(len(hits), 8)] {
				e.Candidates = append(e.Candidates, map[string]any{"id": h.ID, "name": h.Name, "type": h.Type})
			}
			e.Hint = "Retry with an exact name or id."
			return treeNode{}, e
		}
		return treeNode{}, notFound("passive node", name, suggest(name, nodeNames(list), 5), "Retry with one of did_you_mean, or search_data kind=node.")
	}
	var same []treeNode
	for _, h := range hits {
		if norm(h.Name) == norm(hits[0].Name) {
			same = append(same, h)
		}
	}
	if len(same) == 1 || plan == nil {
		return same[0], nil
	}
	best, bestCost := same[0], 1<<30
	for _, c := range same {
		if p := plan.path(c.ID); p != nil && len(p) < bestCost {
			best, bestCost = c, len(p)
		}
	}
	return best, nil
}

type treeOp struct {
	Op             string          `json:"op"`
	Node           json.RawMessage `json:"node"`
	Effect         string          `json:"effect"`
	Path           *bool           `json:"path"`
	WithDependents *bool           `json:"with_dependents"`
	Class          string          `json:"class"`
	Ascendancy     string          `json:"ascendancy"`
}

type editTreeIn struct {
	Handle string   `json:"handle"`
	DryRun bool     `json:"dry_run"`
	Watch  []string `json:"watch"`
	Ops    []treeOp `json:"ops"`
}

var treeOps = []string{"allocate", "deallocate", "set_mastery", "set_class", "set_ascendancy", "reset"}

func checkOps(n, maxOps int) *ToolError {
	if n == 0 {
		return invalidArg("ops must hold at least one op.", nil)
	}
	if n > maxOps {
		return invalidArg(fmt.Sprintf("ops holds at most %d ops.", maxOps), nil)
	}
	return nil
}

func (s *Server) editTree(args json.RawMessage) (any, *ToolError) {
	var in editTreeIn
	if e := decode(args, schemaEditTree, &in); e != nil {
		return nil, e
	}
	if e := checkOps(len(in.Ops), 20); e != nil {
		return nil, e
	}
	return s.runEdit(in.Handle, in.DryRun, in.Watch, len(in.Ops), func(ec *editCtx, i int) *ToolError {
		return applyTreeOp(ec, in.Ops[i])
	})
}

func applyTreeOp(ec *editCtx, op treeOp) *ToolError {
	b := ec.b
	v := specTreeVersion(b)
	switch op.Op {
	case "allocate":
		plan := planPaths(b, v)
		n, e := resolveNode(b, v, op.Node, &plan)
		if e != nil {
			return e
		}
		return allocateNode(ec, n, op.Path == nil || *op.Path, plan)
	case "deallocate":
		n, e := resolveNode(b, v, op.Node, nil)
		if e != nil {
			return e
		}
		if !slices.Contains(b.Build.PassiveNodes, n.ID) {
			return errf(CodeInvalidValue, "%s (id %d) is not allocated.", n.Name, n.ID)
		}
		cut := []int64{n.ID}
		if op.WithDependents == nil || *op.WithDependents {
			cut = append(cut, data.TreeVersions[v].CalculateDependentNodes(slices.Clone(b.Build.PassiveNodes), n.ID, classRoots(b, v))...)
		}
		b.DeallocateNodes(cut)
		if len(cut) > 1 {
			ec.note(fmt.Sprintf("deallocate %s (id %d) and %d dependent nodes", n.Name, n.ID, len(cut)-1))
		} else {
			ec.note(fmt.Sprintf("deallocate %s (id %d)", n.Name, n.ID))
		}
	case "set_mastery":
		plan := planPaths(b, v)
		n, e := resolveNode(b, v, op.Node, &plan)
		if e != nil {
			return e
		}
		if n.Type != "mastery" {
			return errf(CodeInvalidValue, "%s (id %d) is not a mastery.", n.Name, n.ID)
		}
		eff, e := resolveMasteryEffect(n, op.Effect)
		if e != nil {
			return e
		}
		if !slices.Contains(b.Build.PassiveNodes, n.ID) {
			if e := allocateNode(ec, n, true, plan); e != nil {
				return e
			}
		}
		b.SetMasteryEffect(n.ID, eff.Effect)
		ec.note(fmt.Sprintf("set_mastery %s: %s", n.Name, strings.Join(eff.Stats, " / ")))
	case "set_class":
		class, e := resolveClass(op.Class)
		if e != nil {
			return e
		}
		before := len(b.Build.PassiveNodes)
		calculator.SelectClass(b, class)
		ec.note(fmt.Sprintf("set_class %s (%d of %d allocated nodes kept)", class, len(b.Build.PassiveNodes), before))
	case "set_ascendancy":
		asc, e := resolveAscendancy(b.Build.ClassName, op.Ascendancy)
		if e != nil {
			return e
		}
		calculator.SelectAscendClass(b, asc)
		ec.note("set_ascendancy " + string(asc))
	case "reset":
		starts := startNodes(b, v)
		var drop []int64
		for _, id := range b.Build.PassiveNodes {
			if !slices.Contains(starts, id) {
				drop = append(drop, id)
			}
		}
		b.DeallocateNodes(drop)
		ec.note(fmt.Sprintf("reset the tree (%d nodes removed)", len(drop)))
	default:
		return invalidArg(fmt.Sprintf("Unknown op %q.", op.Op), treeOps)
	}
	return nil
}

func allocateNode(ec *editCtx, n treeNode, withPath bool, plan pathPlan) *ToolError {
	b := ec.b
	if n.Ascendancy != "" && n.Ascendancy != string(b.Build.AscendClassName) {
		e := errf(CodeNotReachable, "%s belongs to the %s ascendancy; this build is %s.", n.Name, n.Ascendancy, b.Build.AscendClassName)
		e.Reason = "wrong ascendancy"
		e.Hint = "Use edit_tree op=set_ascendancy first."
		return e
	}
	if slices.Contains(b.Build.PassiveNodes, n.ID) {
		ec.note(fmt.Sprintf("%s (id %d) was already allocated", n.Name, n.ID))
		return nil
	}
	ids := []int64{n.ID}
	if withPath {
		p := plan.path(n.ID)
		if p == nil {
			e := errf(CodeNotReachable, "%s (id %d) has no path from the allocated tree.", n.Name, n.ID)
			e.Reason = "no connected path (cluster subgraph, another class's ascendancy, or a blocked start)"
			return e
		}
		ids = p
	}
	b.AllocateNodes(ids)
	if len(ids) > 1 {
		ec.note(fmt.Sprintf("allocate %s (id %d) via %d path nodes", n.Name, n.ID, len(ids)-1))
	} else {
		ec.note(fmt.Sprintf("allocate %s (id %d)", n.Name, n.ID))
	}
	if n.Type == "mastery" {
		ec.hints = append(ec.hints, "A mastery counts no points until an effect is picked: edit_tree op=set_mastery node=<id> effect=<text>.")
	}
	return nil
}

func resolveMasteryEffect(n treeNode, effect string) (data.MasteryEffect, *ToolError) {
	effs := n.Node.MasteryEffects
	text := func(m data.MasteryEffect) string { return strings.Join(m.Stats, " / ") }
	if id, err := strconv.ParseInt(strings.TrimSpace(effect), 10, 64); err == nil {
		for _, m := range effs {
			if m.Effect == id {
				return m, nil
			}
		}
	}
	if m, ok, _ := resolveName(effect, effs, text, nil); ok && effect != "" {
		return m, nil
	}
	e := errf(CodeNotFound, "%s has no effect %q.", n.Name, effect)
	for _, m := range effs {
		e.Candidates = append(e.Candidates, map[string]any{"id": m.Effect, "text": text(m)})
	}
	e.Hint = "Retry with one of the candidate ids."
	return data.MasteryEffect{}, e
}
