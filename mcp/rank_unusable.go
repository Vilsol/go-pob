package mcp

import (
	"sort"
	"strings"
)

// unusableFlags are the flag prefixes that make a change unusable as it stands, and what each says; an empty why says
// the warning itself: an attribute requirement left unmet, or too little unreserved life or mana for a skill's cost.
var unusableFlags = []struct{ prefix, why string }{
	{"disables: ", "disables a skill"},
	{"removes Weapon 2: ", "drops the off-hand"},
	{"totems: limit ", "lowers the totem limit"},
	{"sockets: ", "overfills sockets"},
	{"warning: You do not meet the ", ""},
	{"warning: You do not have enough ", ""},
	{"warning: Your unreserved Life is below 1", ""},
	{"warning: Your unreserved Mana is negative", ""},
}

// markUnusable marks the rows the character cannot use as they stand, from the flags annotate put on them: a change
// that disables a skill, drops the off-hand, lowers the totem limit, overfills sockets, or leaves an attribute
// requirement unmet or too little life or mana to pay a skill's cost. A loss is big_loss, never unusable. Marked rows
// carry dirtyKey, so they rank after the clean ones, and unusable says why.
func markUnusable(rows []map[string]any) {
	for _, r := range rows {
		var why []string
		flags, _ := r["flags"].([]string)
		for _, u := range unusableFlags {
			for _, f := range flags {
				if !strings.HasPrefix(f, u.prefix) {
					continue
				}
				if u.why != "" {
					why = append(why, u.why)
					break
				}
				why = append(why, strings.TrimPrefix(f, "warning: "))
			}
		}
		if len(why) > 0 {
			r[dirtyKey] = true
			r["unusable"] = strings.Join(why, ", ")
		}
	}
}

// sortUnusableLast moves rows markUnusable marked after the clean ones, keeping each side's order.
func sortUnusableLast(rows []map[string]any) {
	sort.SliceStable(rows, func(i, j int) bool { return rows[i][dirtyKey] == nil && rows[j][dirtyKey] != nil })
}
