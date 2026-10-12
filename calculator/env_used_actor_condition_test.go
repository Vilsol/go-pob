package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestActorConditionUsedRoutesNonEnemyToPlayer pins Calcs.lua:553-558.
func TestActorConditionUsedRoutesNonEnemyToPlayer(t *testing.T) {
	cases := []struct {
		actor      string
		wantPlayer int
		wantEnemy  int
	}{
		{actor: "enemy", wantPlayer: 0, wantEnemy: 1},
		{actor: "parent", wantPlayer: 1, wantEnemy: 0},
		{actor: "player", wantPlayer: 1, wantEnemy: 0},
	}

	for _, c := range cases {
		t.Run(c.actor, func(t *testing.T) {
			env := &Environment{}
			env.initUsedTables()
			m := mod.NewFloat("Damage", mod.TypeIncrease, 10).
				Source("Tree:12345").
				Tag(mod.ActorCondition(c.actor, "Chilled"))
			env.addModTagsUsed(true, m)

			if n := len(env.ConditionsUsed["Chilled"]); n != c.wantPlayer {
				t.Errorf("conditionsUsed[Chilled] = %d, want %d", n, c.wantPlayer)
			}
			if n := len(env.EnemyConditionsUsed["Chilled"]); n != c.wantEnemy {
				t.Errorf("enemyConditionsUsed[Chilled] = %d, want %d", n, c.wantEnemy)
			}
		})
	}
}

// TestEnemyLoopActorConditionRoutingIsInverted pins Calcs.lua:610-615.
func TestEnemyLoopActorConditionRoutingIsInverted(t *testing.T) {
	cases := []struct {
		actor      string
		wantPlayer int
		wantEnemy  int
	}{
		{actor: "enemy", wantPlayer: 1, wantEnemy: 0},
		{actor: "player", wantPlayer: 1, wantEnemy: 0},
		{actor: "parent", wantPlayer: 0, wantEnemy: 1},
	}

	for _, c := range cases {
		t.Run(c.actor, func(t *testing.T) {
			db := moddb.NewModDB()
			db.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 10).
				Source("Tree:12345").
				Tag(mod.ActorCondition(c.actor, "Chilled")))
			env := &Environment{EnemyModDB: db}
			env.initUsedTables()
			collectEnemyTagsUsed(env)

			if n := len(env.ConditionsUsed["Chilled"]); n != c.wantPlayer {
				t.Errorf("conditionsUsed[Chilled] = %d, want %d", n, c.wantPlayer)
			}
			if n := len(env.EnemyConditionsUsed["Chilled"]); n != c.wantEnemy {
				t.Errorf("enemyConditionsUsed[Chilled] = %d, want %d", n, c.wantEnemy)
			}
		})
	}
}
