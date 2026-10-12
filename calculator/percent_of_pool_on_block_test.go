package calculator

import (
	"math"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestRecoverPercentOfPoolCeils pins ModStore.lua:532 through ModParser.lua:4711-4717.
func TestRecoverPercentOfPoolCeils(t *testing.T) {
	for _, tc := range []struct {
		name    string
		life    float64
		percent float64
		want    float64
	}{
		{"10327 at 3%", 10327, 3, 310},
		{"12862 at 5%", 12862, 5, 644},
		{"18817 at 5%", 18817, 5, 941},
		{"10000 at 3% (exact)", 10000, 3, 300},
		{"10001 at 3%", 10001, 3, 301},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := moddb.NewModDB()
			cfg := &moddb.ListCfg{SkillStats: map[string]float64{"Life": tc.life}}
			db.AddMod(mod.NewFloat("LifeOnBlock", mod.TypeBase, 1).Tag(mod.PercentStat("Life", tc.percent)))
			if got := db.Sum(mod.TypeBase, cfg, "LifeOnBlock"); got != tc.want {
				t.Fatalf("got %v, want %v (ceil of %v)", got, tc.want, tc.life*tc.percent/100)
			}
		})
	}
}

// TestRecoverPercentOfPoolRowsUseTheCeilingTag is the ROW-level half.
func TestRecoverPercentOfPoolRowsUseTheCeilingTag(t *testing.T) {
	for _, tc := range []struct {
		line string
		name string
		stat string
	}{
		{"Recover 3% of Life when you Block", "LifeOnBlock", "Life"},
		{"Recover 4% of your maximum Mana when you Block", "ManaOnBlock", "Mana"},
		{"Recover 5% of Energy Shield when you Block", "EnergyShieldOnBlock", "EnergyShield"},
		{"Recover Energy Shield equal to 6% of Armour when you Block", "EnergyShieldOnBlock", "Armour"},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if len(e.ModList) != 1 {
				t.Fatalf("got %d mods, want 1 (extra %q)", len(e.ModList), e.Extra)
			}
			m := e.ModList[0]
			if m.Name() != tc.name {
				t.Fatalf("got %s, want %s", m.Name(), tc.name)
			}
			if len(m.Tags()) != 1 {
				t.Fatalf("got %d tags, want 1", len(m.Tags()))
			}
			ps, ok := m.Tags()[0].(*mod.PercentStatTag)
			if !ok {
				t.Fatalf("tag is %T, want *mod.PercentStatTag — a PerStat tag FLOORS where PoB ceils", m.Tags()[0])
			}
			if ps.Stat != tc.stat {
				t.Fatalf("tag reads %q, want %q", ps.Stat, tc.stat)
			}
			if math.Abs(m.Value().Float()-1) > 1e-9 {
				t.Fatalf("mod value is %v, want 1", m.Value().Float())
			}
		})
	}
}
