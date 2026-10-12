package raw

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestMultiplierStatMapsCarryTheLateFields pins the eighteen skill-statMap mods whose Multiplier tag carries one of the four fields added to the tag between v2.18.1 and the 3.29 pin.
func TestMultiplierStatMapsCarryTheLateFields(t *testing.T) {
	noFloor, divVar, limitNegTotal, limitStat := 0, 0, 0, 0
	for _, perSkill := range SkillStatMaps {
		for _, entry := range perSkill {
			for _, m := range entry.Mods {
				for _, raw := range m.Tags() {
					mt, ok := raw.(*mod.MultiplierTag)
					if !ok {
						continue
					}
					if mt.TagNoFloor {
						noFloor++
					}
					if mt.TagDivVariable != nil {
						divVar++
					}
					if mt.TagLimitNegTotal {
						limitNegTotal++
					}
					if mt.TagLimitStat != nil {
						limitStat++
					}
				}
			}
		}
	}

	for _, c := range []struct {
		name string
		got  int
		want int
	}{
		{"noFloor", noFloor, 15},
		{"divVar", divVar, 2},
		{"limitNegTotal", limitNegTotal, 1},
		{"limitStat", limitStat, 0},
	} {
		if c.got != c.want {
			t.Errorf("%s: %d Multiplier tags carry it, want %d — the emitter allowlist or the "+
				"pinned skill data has moved", c.name, c.got, c.want)
		}
	}
}
