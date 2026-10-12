package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

var enemyModifierLines = []string{
	"20% increased Blind Effect",
	"Hits that deal Elemental Damage remove Exposure to those Elements and inflict Exposure to other Elements Exposure inflicted this way applies -13% to Resistances",
	"Enemies you Hit with Elemental Damage temporarily get +25% Resistance to those Elements and -50% Resistance to other Elements",
}

// TestEnemyModifierListValueShape pins the payload every "EnemyModifier" LIST mod must carry.
func TestEnemyModifierListValueShape(t *testing.T) {
	for _, line := range enemyModifierLines {
		entry := ParseMod(line, false)
		if entry.ModList == nil {
			t.Fatalf("%q did not parse", line)
		}

		found := 0
		for _, m := range entry.ModList {
			if m.Name() != "EnemyModifier" {
				continue
			}
			found++
			if _, ok := m.Value().List().(mod.EnemyModifier); !ok {
				t.Errorf("%q: EnemyModifier list value is %T, want mod.EnemyModifier", line, m.Value().List())
			}
		}
		if found == 0 {
			t.Errorf("%q produced no EnemyModifier mod", line)
		}
	}
}

// TestEnemyModifierAppliesToEnemy runs the DoActorMisc consumer loop over a ModDB fed by the parser.
func TestEnemyModifierAppliesToEnemy(t *testing.T) {
	db := moddb.NewModDB()
	for _, line := range enemyModifierLines {
		entry := ParseMod(line, false)
		if entry.ModList == nil {
			t.Fatalf("%q did not parse", line)
		}
		for _, m := range entry.ModList {
			db.AddMod(m)
		}
	}

	enemy := moddb.NewModDB()
	for _, value := range db.List(nil, "EnemyModifier") {
		enemy.AddMod(value.(mod.EnemyModifier).Mod)
	}

	if got := enemy.Sum(mod.TypeIncrease, nil, "BlindEffect"); got != 20 {
		t.Fatalf("enemy BlindEffect = %v, want 20", got)
	}
}
