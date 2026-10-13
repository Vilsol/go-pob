package calculator

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestRenamedFlagsCoverEveryUpstreamRow is the gate for a defect class this repo found the expensive way.
func TestRenamedFlagsCoverEveryUpstreamRow(t *testing.T) {
	src := pinnedPoBSrcOrSkip(t)

	renames := []struct {
		upstream string
		ours     string
		why      string
	}{}

	lua, err := os.ReadFile(filepath.Join(src, "Modules", "ModParser.lua"))
	if err != nil {
		t.Fatalf("reading the pin's ModParser.lua: %v", err)
	}
	parser, err := os.ReadFile("mod_parser.go")
	if err != nil {
		t.Fatalf("reading mod_parser.go: %v", err)
	}
	ourText := string(parser)

	for _, r := range renames {
		t.Run(r.upstream, func(t *testing.T) {
			row := regexp.MustCompile(`\["([^"]+)"\][^\n]*\bflag\("` + regexp.QuoteMeta(r.upstream) + `"\)`)
			matches := row.FindAllStringSubmatch(string(lua), -1)

			if len(matches) == 0 {
				t.Fatalf("REFUSING: the pin's ModParser.lua emits no flag(%q) at all. Either it was "+
					"renamed upstream — in which case %q is now chasing a name that no longer exists "+
					"— or this test's row pattern has stopped matching. Both are findings; neither is "+
					"coverage.", r.upstream, r.ours)
			}

			var missing []string
			for _, m := range matches {
				phrase := m[1]
				key := strings.ToLower(phrase)
				if strings.Contains(ourText, `"`+key+`"`) || strings.Contains(ourText, "`"+key+"`") {
					continue
				}
				missing = append(missing, phrase)
			}
			sort.Strings(missing)

			if len(missing) > 0 {
				t.Errorf("%s is this tree's name for upstream's %s (%s), and %d of the pin's %d rows "+
					"for that flag have NO row here: %q.\n"+
					"That is the CanLeechEnergyShieldOnFullEnergyShield shape: the pair is internally "+
					"consistent, no value is wrong, and the upstream name having no hits in our tree is "+
					"the only tell. Either add the missing rows under %s, or rename ours to %s so the "+
					"next census can see them.",
					r.ours, r.upstream, r.why, len(missing), len(matches), missing, r.ours, r.upstream)
			}
		})
	}
}
