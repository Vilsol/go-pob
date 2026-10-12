package calculator

import "testing"

// TestPassiveSkillsInRadiusWording pins ModParser.lua:5154-5161, whose keys are `passives?
func TestPassiveSkillsInRadiusWording(t *testing.T) {
	for _, tc := range []struct {
		line string
		mods int
	}{
		{"Passives in radius can be allocated without being connected to your tree", 1},
		{"Passive Skills in Radius can be Allocated without being connected to your tree", 1},
		{"Passives in radius of Pain Attunement can be allocated without being connected to your tree", 2},
		{"Passive Skills in radius of Pain Attunement can be allocated without being connected to your tree", 2},
		{"Keystone Passive Skills in Radius can be Allocated without being connected to your tree", 2},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil {
				t.Fatalf("ParseMod returned nil")
			}
			if e.Extra != "" {
				t.Errorf("leftover Extra %q", e.Extra)
			}
			if len(e.ModList) != tc.mods {
				t.Fatalf("got %d mods, want %d: %v", len(e.ModList), tc.mods, e.ModList)
			}
			for _, m := range e.ModList {
				if m.Name() != "JewelData" && m.Name() != "ImpossibleEscapeKeystones" {
					t.Errorf("unexpected mod name %q", m.Name())
				}
			}
		})
	}
}
