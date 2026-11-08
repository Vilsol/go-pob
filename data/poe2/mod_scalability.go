package poe2

import "github.com/Vilsol/go-pob-data/poe2/pobdata"

// ModScalabilityEntry is one data.modScalability entry: entry i governs the line's i-th `#`.
//
//pob2:block:src/Modules/ItemTools.lua:275-279
type ModScalabilityEntry = pobdata.ModScalabilityEntry

func modScalabilityByLine(rows []pobdata.ModScalability) map[string][]ModScalabilityEntry {
	m := make(map[string][]ModScalabilityEntry, len(rows))
	for _, row := range rows {
		if row.Entries == nil {
			m[row.Line] = []ModScalabilityEntry{}
			continue
		}
		m[row.Line] = row.Entries
	}
	return m
}

// ModScalability is data.modScalability[key]; ok is the key's presence, which an empty list still has.
//
//pob2:block:src/Modules/ItemTools.lua:197-197
func ModScalability(key string) ([]ModScalabilityEntry, bool) {
	entries, ok := itemTables().modScalability[key]
	return entries, ok
}
