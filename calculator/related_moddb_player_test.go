package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/moddb"
)

// TestRelatedModDBResolvesPlayerThreeWays pins ModStore.lua:37-43's getActor.
func TestRelatedModDBResolvesPlayerThreeWays(t *testing.T) {
	newActor := func() *Actor { return &Actor{ModDB: moddb.NewModDB()} }

	player := newActor()
	parent := newActor()
	parent.Player = player
	enemy := newActor()
	enemy.Player = player

	for _, tc := range []struct {
		name  string
		actor func() *Actor
		want  *Actor
	}{
		{"direct: actor.player", func() *Actor { a := newActor(); a.Player = player; return a }, player},
		{"via parent: actor.parent.player", func() *Actor { a := newActor(); a.Parent = parent; return a }, player},
		{"via enemy: actor.enemy.player", func() *Actor { a := newActor(); a.Enemy = enemy; return a }, player},
		{
			"direct beats parent and enemy",
			func() *Actor {
				a := newActor()
				a.Player = player
				other := newActor()
				a.Parent = &Actor{ModDB: moddb.NewModDB(), Player: other}
				a.Enemy = &Actor{ModDB: moddb.NewModDB(), Player: other}
				return a
			},
			player,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.actor().RelatedModDB("player")
			if got == nil {
				t.Fatal("RelatedModDB(\"player\") = nil — the arm is absent, and a tag carrying " +
					"actor = \"player\" is then dropped on every actor")
			}
			if got != moddb.ModStoreFuncs(tc.want.ModDB) {
				t.Errorf("RelatedModDB(\"player\") resolved to the wrong actor's ModDB")
			}
		})
	}

	if got := newActor().RelatedModDB("player"); got != nil {
		t.Errorf("RelatedModDB(\"player\") on an unlinked actor = %v, want nil", got)
	}
}
