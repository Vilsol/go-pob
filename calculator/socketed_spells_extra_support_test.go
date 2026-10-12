package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestSocketedSpellsSupportedByLevelRow pins ModParser.lua:3372, the `spells` sibling of the `gems` row at :3371.
func TestSocketedSpellsSupportedByLevelRow(t *testing.T) {
	for _, tc := range []struct {
		name string
		line string
	}{
		{"spells (the ported row)", "Socketed Spells are Supported by level 20 Crab Totem"},
		{"gems (control, passes in both states)", "Socketed Gems are Supported by level 20 Crab Totem"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil || len(e.ModList) == 0 {
				t.Fatalf("ParseMod(%q) produced no mods", tc.line)
			}
			if e.Extra != "" {
				t.Errorf("leftover Extra %q — processModLine discards the whole line", e.Extra)
			}
			found := false
			for _, m := range e.ModList {
				if m.Name() != "ExtraSupport" {
					continue
				}
				es, ok := m.Value().List().(mod.ExtraSupport)
				if !ok {
					if p, okp := m.Value().List().(*mod.ExtraSupport); okp {
						es, ok = *p, true
					}
				}
				if !ok {
					t.Fatalf("ExtraSupport payload has unexpected type %T", m.Value().List())
				}
				if es.SkillID == "SupportCrabTotem" {
					found = true
					if es.Level != 20 {
						t.Errorf("level = %d, want 20", es.Level)
					}
				}
			}
			if !found {
				t.Errorf("no ExtraSupport{SkillID: SupportCrabTotem} in %v", e.ModList)
			}
		})
	}
}
