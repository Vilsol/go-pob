package calculator

import (
	"sync"

	"github.com/Vilsol/go-pob/utils"
)

func fixupTreeStatText(stats []string) []string {
	var out []string
	for i, line := range stats {
		fixed := fixedTreeStatLine(line)
		if fixed == line {
			continue
		}
		if out == nil {
			out = make([]string, len(stats))
			copy(out, stats)
		}
		out[i] = fixed
	}
	if out == nil {
		return stats
	}
	return out
}

var fixedTreeStatLines sync.Map

func fixedTreeStatLine(line string) string {
	if fixed, ok := fixedTreeStatLines.Load(line); ok {
		return fixed.(string)
	}
	fixed := utils.EscapeGGGString(line)
	if f, ok := treeStatTextFixups[fixed]; ok {
		fixed = f
	}
	fixedTreeStatLines.Store(line, fixed)
	return fixed
}
