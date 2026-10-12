package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestCalcActionSpeedModClamps: CalcPerform.lua:1107-1117 has TWO clamps on top of the increase sum.
func TestCalcActionSpeedModClamps(t *testing.T) {
	newActor := func(mods ...mod.Mod) *Actor {
		db := moddb.NewModDB()
		for _, m := range mods {
			db.AddMod(m)
		}
		return &Actor{ModDB: db}
	}

	t.Run("no mods is unmodified", func(t *testing.T) {
		if got := CalcActionSpeedMod(newActor()); got != 1 {
			t.Fatalf("want 1, got %v", got)
		}
	})

	t.Run("a slow with no floor applies", func(t *testing.T) {
		a := newActor(mod.NewFloat("ActionSpeed", mod.TypeIncrease, -50))
		if got := CalcActionSpeedMod(a); got != 0.5 {
			t.Fatalf("want 0.5, got %v", got)
		}
	})

	//pob:src/Modules/CalcPerform.lua:1109
	//pob:src/Modules/CalcPerform.lua:1112
	t.Run("MinimumActionSpeed floors the slow", func(t *testing.T) {
		a := newActor(
			mod.NewFloat("ActionSpeed", mod.TypeIncrease, -50),
			mod.NewFloat("MinimumActionSpeed", mod.TypeMAX, 100),
		)
		if got := CalcActionSpeedMod(a); got != 1 {
			t.Fatalf("want 1, got %v", got)
		}
	})

	t.Run("MinimumActionSpeed floors partially", func(t *testing.T) {
		a := newActor(
			mod.NewFloat("ActionSpeed", mod.TypeIncrease, -60),
			mod.NewFloat("MinimumActionSpeed", mod.TypeMAX, 70),
		)
		if got := CalcActionSpeedMod(a); got != 0.7 {
			t.Fatalf("want 0.7, got %v", got)
		}
	})

	t.Run("MinimumActionSpeed does not raise an unslowed actor", func(t *testing.T) {
		a := newActor(
			mod.NewFloat("ActionSpeed", mod.TypeIncrease, 50),
			mod.NewFloat("MinimumActionSpeed", mod.TypeMAX, 100),
		)
		if got := CalcActionSpeedMod(a); got != 1.5 {
			t.Fatalf("want 1.5, got %v", got)
		}
	})

	//pob:src/Modules/CalcPerform.lua:1110
	//pob:src/Modules/CalcPerform.lua:1113-1115
	t.Run("MaximumActionSpeedReduction caps the increase", func(t *testing.T) {
		a := newActor(
			mod.NewFloat("ActionSpeed", mod.TypeIncrease, 50),
			mod.NewFloat("MaximumActionSpeedReduction", mod.TypeMAX, 20),
		)
		if got := CalcActionSpeedMod(a); got != 0.8 {
			t.Fatalf("want 0.8, got %v", got)
		}
	})

	t.Run("MaximumActionSpeedReduction does not lift a deeper slow", func(t *testing.T) {
		a := newActor(
			mod.NewFloat("ActionSpeed", mod.TypeIncrease, -50),
			mod.NewFloat("MaximumActionSpeedReduction", mod.TypeMAX, 20),
		)
		if got := CalcActionSpeedMod(a); got != 0.5 {
			t.Fatalf("want 0.5, got %v", got)
		}
	})

	t.Run("the parsed Juggernaut line floors at base", func(t *testing.T) {
		a := newActor(
			mod.NewFloat("ActionSpeed", mod.TypeIncrease, -50),
			ParseMod("Action Speed cannot be modified to below Base Value", false).ModList[0],
		)
		if got := CalcActionSpeedMod(a); got != 1 {
			t.Fatalf("want 1, got %v", got)
		}
	})
}
