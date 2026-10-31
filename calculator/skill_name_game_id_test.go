package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestSkillNameTransfiguredResolvesLikeThePin pins ModStore.lua:763-774 with calcLib.getGameIdFromGemName(cfg.skillName.
func TestSkillNameTransfiguredResolvesLikeThePin(t *testing.T) {
	tag := func(name string) mod.Tag { return mod.SkillName(name).Transfigured(true) }
	for _, tc := range []struct {
		name  string
		tag   mod.Tag
		cfg   *moddb.ListCfg
		match bool
	}{
		{"vaal-haste", tag("Haste"), &moddb.ListCfg{SkillName: "Haste", SkillGrantedEffect: gameIDGrantedEffect{id: "VaalHaste"}}, true},
		{"vaal-fireball", tag("Fireball"), &moddb.ListCfg{SkillName: "Fireball", SkillGrantedEffect: gameIDGrantedEffect{id: "VaalFireball"}}, true},
		{"transfigured", tag("Lightning Tendrils"), &moddb.ListCfg{SkillName: "Lightning Tendrils of Eccentricity", SkillGrantedEffect: gameIDGrantedEffect{id: "LightningTendrilsAltX"}}, true},
		{"plain", tag("Frenzy"), &moddb.ListCfg{SkillName: "Frenzy", SkillGrantedEffect: gameIDGrantedEffect{id: "Frenzy"}}, true},
		{"other-skill", tag("Haste"), &moddb.ListCfg{SkillName: "Frenzy", SkillGrantedEffect: gameIDGrantedEffect{id: "Frenzy"}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := moddb.NewModDB()
			db.Actor = gameIDActor{}
			db.AddMod(mod.NewFloat("AuraEffect", mod.TypeIncrease, 10).Tag(tc.tag))
			got := db.Sum(mod.TypeIncrease, tc.cfg, "AuraEffect") == 10
			if got != tc.match {
				t.Errorf("match = %v, want %v", got, tc.match)
			}
		})
	}
}

type gameIDActor struct{}

func (gameIDActor) GetOutput(string) (float64, bool) { return 0, false }

type gameIDGrantedEffect struct{ id string }

func (g gameIDGrantedEffect) ID() string { return g.id }
