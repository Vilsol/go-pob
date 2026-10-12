package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestConsumerBackedItemRowsParse pins six ModParser.lua rows whose target mod name ALREADY had a reader in this tree and no producer anywhere.
func TestConsumerBackedItemRowsParse(t *testing.T) {
	for _, tc := range []struct {
		line  string
		names []string
		types []mod.Type
		vals  []float64
	}{
		{"Your Warcries cover Enemies in Ash for 5 seconds", []string{"CoveredInAshEffect"}, []mod.Type{mod.TypeBase}, []float64{20}},
		{"Debuffs on you expire 18% faster while affected by Haste", []string{"SelfDebuffExpirationRate"}, []mod.Type{mod.TypeBase}, []float64{18}},
		{"15% of Damage taken while affected by Clarity Recouped as Mana", []string{"ManaRecoup"}, []mod.Type{mod.TypeBase}, []float64{15}},
		{"+1 to maximum number of Sacred Wisps", []string{"SacredWispsMaxCount"}, []mod.Type{mod.TypeBase}, []float64{1}},
		{"+8 to maximum number of Eaten Souls", []string{"SoulEaterMax"}, []mod.Type{mod.TypeBase}, []float64{8}},
		{"Maximum 30 Eaten Souls", []string{"SoulEaterMax"}, []mod.Type{mod.TypeOverride}, []float64{30}},
		{"Instant Recovery when on Low Life", []string{"FlaskLowLifeInstantRecovery", "Dummy"}, []mod.Type{mod.TypeBase, mod.TypeDummy}, []float64{100, 1}},
		{"Instant Recovery", []string{"FlaskInstantRecovery"}, []mod.Type{mod.TypeBase}, []float64{100}},
		{"Debuffs on you expire 12% faster", []string{"SelfDebuffExpirationRate"}, []mod.Type{mod.TypeBase}, []float64{12}},
		{"Nearby Enemies are Covered in Ash", []string{"CoveredInAshEffect"}, []mod.Type{mod.TypeBase}, []float64{20}},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil {
				t.Fatalf("ParseMod(%q) returned nil", tc.line)
			}
			if e.Extra != "" {
				t.Errorf("leftover Extra %q — processModLine discards the whole line", e.Extra)
			}
			if len(e.ModList) != len(tc.names) {
				got := make([]string, 0, len(e.ModList))
				for _, m := range e.ModList {
					got = append(got, m.Name())
				}
				t.Fatalf("mods = %v, want %v", got, tc.names)
			}
			for i, m := range e.ModList {
				if m.Name() != tc.names[i] {
					t.Errorf("mod[%d] name = %q, want %q", i, m.Name(), tc.names[i])
				}
				if m.Type() != tc.types[i] {
					t.Errorf("mod[%d] %s type = %v, want %v", i, m.Name(), m.Type(), tc.types[i])
				}
				if v := m.Value().Float(); v != tc.vals[i] {
					t.Errorf("mod[%d] %s value = %v, want %v", i, m.Name(), v, tc.vals[i])
				}
			}
		})
	}
}
