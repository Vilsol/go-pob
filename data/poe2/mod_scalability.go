package poe2

import (
	"strings"
	"sync"
)

// ModScalabilityEntry is one data.modScalability entry: entry i governs the line's i-th `#`.
//
//pob2:block:src/Modules/ItemTools.lua:275-279
type ModScalabilityEntry struct {
	IsScalable bool
	Formats    []string
}

var modScalability = sync.OnceValue(func() map[string][]ModScalabilityEntry {
	m := make(map[string][]ModScalabilityEntry, len(modScalabilityRows))
	for _, row := range modScalabilityRows {
		key, body, _ := strings.Cut(row, "\x1f")
		if body == "" {
			m[key] = []ModScalabilityEntry{}
			continue
		}
		parts := strings.Split(body, "\x1d")
		entries := make([]ModScalabilityEntry, len(parts))
		for i, part := range parts {
			entries[i].IsScalable = part[0] == '1'
			if len(part) > 1 {
				entries[i].Formats = strings.Split(part[2:], "\x1c")
			}
		}
		m[key] = entries
	}
	return m
})

// ModScalability is data.modScalability[key]; ok is the key's presence, which an empty list still has.
//
//pob2:block:src/Modules/ItemTools.lua:197-197
func ModScalability(key string) ([]ModScalabilityEntry, bool) {
	entries, ok := modScalability()[key]
	return entries, ok
}
