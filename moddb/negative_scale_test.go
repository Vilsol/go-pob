package moddb

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestScaleAddListKeepsNegativeScales pins ModStore.lua:45-86, which has NO clamp on `scale`.
func TestScaleAddListKeepsNegativeScales(t *testing.T) {
	for _, tc := range []struct {
		name  string
		scale float64
		start float64
		want  float64
	}{
		{"negative scale truncates toward zero", -0.2, 26, -5},
		{"positive scale truncates toward zero", 0.2, 26, 5},

		{"CONTROL positive scale unaffected by the clamp", 0.5, 10, 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := NewModList()
			src.AddMod(mod.NewFloat("Life", mod.TypeBase, tc.start))

			dst := NewModList()
			dst.ScaleAddList(src, tc.scale)

			mods := dst.Mods()
			if len(mods) != 1 {
				t.Fatalf("got %d mods, want 1", len(mods))
			}
			if got := mods[0].Value().Float(); got != tc.want {
				t.Errorf("scaled value = %v, want %v", got, tc.want)
			}
			if got := src.Mods()[0].Value().Float(); got != tc.start {
				t.Errorf("source mutated: %v, want %v", got, tc.start)
			}
		})
	}
}
