package calculator

import (
	"slices"
	"sort"
	"strconv"

	"github.com/Vilsol/go-pob/data"
)

type SpecNodeRewrite struct {
	NodeID   int64
	Name     string
	Stats    []string
	Replaced bool
}

//pob:block:src/Classes/PassiveSpec.lua:1614-1635,src/Classes/PassiveSpec.lua:2380-2388
func SpecNodeRewrites(env *Environment) []SpecNodeRewrite {
	tree := data.TreeVersions[data.LatestTreeVersion].Tree()
	if env.Spec != nil {
		tree = env.Spec.Tree()
	}
	vaal := vaalConquered(env, tree)
	var out []SpecNodeRewrite
	for id, r := range nodeRewrites(env) {
		n, ok := tree.Nodes[id]
		if !ok {
			continue
		}
		nodeID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			continue
		}
		rw := SpecNodeRewrite{NodeID: nodeID, Stats: n.Stats}
		if n.Name != nil {
			rw.Name = *n.Name
		}
		if rep := r.effectiveReplacement(); rep != nil {
			rw.Name, rw.Stats, rw.Replaced = rep.DN, rep.Stats, true
		}
		for _, line := range r.addStats {
			if line == "" {
				continue
			}
			if !vaal[id] {
				rw.Stats = append(slices.Clip(rw.Stats), " ")
			}
			rw.Stats = append(slices.Clip(rw.Stats), line)
		}
		if !rw.Replaced && len(r.addStats) == 0 {
			continue
		}
		out = append(out, rw)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NodeID < out[j].NodeID })
	return out
}

func vaalConquered(env *Environment, tree *data.Tree) map[string]bool {
	jewels := socketedLegionJewels(env)
	if len(jewels) == 0 {
		return nil
	}
	radiusNodes := tree.JewelRadiusNodes()
	out := map[string]bool{}
	for _, j := range jewels {
		if isExpansionSocket(tree, strconv.FormatInt(j.socketNodeID, 10)) || legionJewelType(j.jewel.Conqueror.Type) >= 7 {
			continue
		}
		perRadius, ok := radiusNodes[j.socketNodeID]
		if !ok || j.radiusIndex < 0 || j.radiusIndex >= len(perRadius) {
			continue
		}
		for _, id := range perRadius[j.radiusIndex] {
			out[strconv.FormatInt(id, 10)] = j.jewel.Conqueror.Type == "vaal"
		}
	}
	return out
}
