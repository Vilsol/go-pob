package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Vilsol/go-pob/utils"
)

const tdUsage = "usage: tree_divergence.py <ours.json> <pob.json>"

const tdRefusalExit = 3

func tdRefuse(msg string) int {
	fmt.Fprintf(os.Stderr, "REFUSING: %s\n", msg)
	return tdRefusalExit
}

type tdDump struct {
	version any
	nodes   map[string]any
	order   []string
}

func tdLoad(path string) (*tdDump, int) {
	body, err := os.ReadFile(path) //nolint:gosec // the dump path is the caller's argument
	if err != nil {
		return nil, tdRefuse(fmt.Sprintf("%s is not readable: %s", path, errnoText(err)))
	}
	m, reason := pyLoadJSONObjectBytes(body, path)
	if reason != "" {
		return nil, tdRefuse(reason)
	}
	nodes, ok := m["nodes"].(map[string]any)
	if !ok {
		return nil, tdRefuse(fmt.Sprintf("%s has no 'nodes' object", path))
	}
	order, ok := pyObjectKeyOrder(body, "nodes")
	if !ok {
		return nil, tdRefuse(fmt.Sprintf("%s has no 'nodes' object", path))
	}
	return &tdDump{version: m["version"], nodes: nodes, order: order}, 0
}

func tdFlatten(stats []any) ([]string, string) {
	out := []string{}
	for _, s := range stats {
		str, ok := s.(string)
		if !ok {
			return nil, fmt.Sprintf("a node's stats hold a %s, not a string", pyTypeName(s))
		}
		out = append(out, strings.Split(str, "\n")...)
	}
	kept := out[:0]
	for _, x := range out {
		if t := strings.TrimSpace(x); t != "" {
			kept = append(kept, t)
		}
	}
	return kept, ""
}

func tdNameOf(nodes map[string]any, key string) (string, string) {
	node, ok := nodes[key].(map[string]any)
	if !ok {
		return "", fmt.Sprintf("node %s has no 'name' string", key)
	}
	name, ok := node["name"].(string)
	if !ok {
		return "", fmt.Sprintf("node %s has no 'name' string", key)
	}
	return name, ""
}

func tdStatsOf(nodes map[string]any, key string) ([]any, string) {
	node, ok := nodes[key].(map[string]any)
	if !ok {
		return nil, fmt.Sprintf("node %s has no 'stats' list", key)
	}
	stats, ok := node["stats"].([]any)
	if !ok {
		return nil, fmt.Sprintf("node %s has no 'stats' list", key)
	}
	return stats, ""
}

const tdTail = "\n⚠️ A DIVERGENCE IS NOT AUTOMATICALLY A FIX TO ADD. Measured 2026-08-09: adding node\n" +
	"   5087 (Born in the Shadows, ours 'Cannot be Blinded' vs PoB '100% chance to Avoid\n" +
	"   Blind') to treeStatTextFixups COSTS 232 pairs — our parser handles our own wording\n" +
	"   better than PoB's. Measure any entry before adding it, and record the number."

func treeDivergence(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, tdUsage)
		return 2
	}
	ours, code := tdLoad(args[0])
	if code != 0 {
		return code
	}
	pob, code := tdLoad(args[1])
	if code != 0 {
		return code
	}
	on, pn := ours.nodes, pob.nodes

	if len(on) == 0 || len(pn) == 0 {
		return tdRefuse("one of the dumps has no nodes")
	}

	w := os.Stdout
	fmt.Fprintf(w, "ours tree %s  %d nodes with stats\n", oeStr(ours.version), len(on))
	fmt.Fprintf(w, "pob  tree %s  %d nodes with stats\n", oeStr(pob.version), len(pn))

	missing := []string{}
	for k := range pn {
		if _, ok := on[k]; !ok {
			missing = append(missing, k)
		}
	}
	sort.Strings(missing)
	fmt.Fprintf(w, "\nNODES PoB HAS AND WE DO NOT: %d\n", len(missing))
	for _, k := range missing {
		name, reason := tdNameOf(pn, k)
		if reason != "" {
			return tdRefuse(reason)
		}
		fmt.Fprintf(w, "  %7s  %s\n", k, name)
	}

	extra := []string{}
	for k := range on {
		if _, ok := pn[k]; ok {
			continue
		}
		node, _ := on[k].(map[string]any)
		stats, _ := node["stats"].([]any)
		if lines, reason := tdFlatten(stats); reason != "" {
			//go-only: the ours-only listing post-dates the python tool; tdFlatten's own refusal.
			return tdRefuse(reason)
		} else if len(lines) > 0 {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	fmt.Fprintf(w, "\nNODES WE HAVE AND PoB DOES NOT (with stats): %d\n", len(extra))
	for _, k := range extra {
		name, reason := tdNameOf(on, k)
		if reason != "" {
			return tdRefuse(reason)
		}
		fmt.Fprintf(w, "  %7s  %s\n", k, name)
	}

	type tdDiff struct {
		key, name string
		ours, pob []string
	}
	diverged := []tdDiff{}
	for _, k := range pob.order {
		if _, ok := on[k]; !ok {
			continue
		}
		os1, reason := tdStatsOf(on, k)
		if reason != "" {
			return tdRefuse(reason)
		}
		ps1, reason := tdStatsOf(pn, k)
		if reason != "" {
			return tdRefuse(reason)
		}
		o, reason := tdFlatten(os1)
		if reason != "" {
			return tdRefuse(reason)
		}
		for i := range o {
			o[i] = utils.EscapeGGGString(o[i])
		}
		p, reason := tdFlatten(ps1)
		if reason != "" {
			return tdRefuse(reason)
		}
		if !tdEqual(o, p) {
			name, reason := tdNameOf(pn, k)
			if reason != "" {
				return tdRefuse(reason)
			}
			diverged = append(diverged, tdDiff{key: k, name: name, ours: o, pob: p})
		}
	}

	fmt.Fprintf(w, "\nSTAT-TEXT DIVERGENCES (line splitting normalised): %d\n", len(diverged))
	for _, d := range diverged {
		fmt.Fprintf(w, "  %7s  %s\n", d.key, d.name)
		for _, a := range d.ours {
			if !tdContains(d.pob, a) {
				fmt.Fprintf(w, "           ours: %s\n", a)
			}
		}
		for _, b := range d.pob {
			if !tdContains(d.ours, b) {
				fmt.Fprintf(w, "           pob : %s\n", b)
			}
		}
	}

	fmt.Fprintln(w, tdTail)
	return 0
}

func tdEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func tdContains(hay []string, needle string) bool {
	for _, x := range hay {
		if x == needle {
			return true
		}
	}
	return false
}
