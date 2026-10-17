package calculator

import (
	"regexp"
	"strings"
	"sync/atomic"

	"github.com/Vilsol/go-pob/mod"
)

var modParserExtEnabled atomic.Bool

func init() {
	modParserExtEnabled.Store(true)
}

func SetModParserExtensions(on bool) bool {
	return modParserExtEnabled.Swap(on)
}

type modParserExt struct {
	id       string
	pattern  *regexp.Regexp
	mods     func(caps []string) []mod.Mod
	examples []string
}

var modParserExtList = []modParserExt{
	//gopob:ext adrenaline-on-kill-singular-second the pin parses "for N seconds on kill" but leaves the singular "second" over, with the same mods
	{
		id:      "adrenaline-on-kill-singular-second",
		pattern: regexp.MustCompile(`^gain adrenaline for (\d+) second on kill$`),
		mods: func(caps []string) []mod.Mod {
			return []mod.Mod{mod.NewFlag("Condition:Adrenaline", true).Tag(mod.Condition("KilledRecently"))}
		},
		examples: []string{"Gain Adrenaline for 2 second on Kill"},
	},
}

func modParserExtUnsupported(entry *ModCacheEntry) bool {
	return entry.ModList == nil || entry.Extra != ""
}

func parseModExt(line string) (string, []mod.Mod) {
	lower := strings.ToLower(strings.TrimSpace(line))
	for _, ext := range modParserExtList {
		if m := ext.pattern.FindStringSubmatch(lower); m != nil {
			return ext.id, ext.mods(m[1:])
		}
	}
	return "", nil
}

func applyModParserExt(line string, entry *ModCacheEntry, on bool) *ModCacheEntry {
	if !on || !modParserExtUnsupported(entry) {
		return entry
	}
	if id, mods := parseModExt(line); id != "" {
		return &ModCacheEntry{ModList: mods, Extra: ""}
	}
	return entry
}
