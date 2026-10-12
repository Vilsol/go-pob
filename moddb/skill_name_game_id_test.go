package moddb

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestSkillNameTransfiguredResolvesLikeThePin pins ModStore.lua:763-774 with calcLib.getGameIdFromGemName(cfg.skillName.
func TestSkillNameTransfiguredResolvesLikeThePin(t *testing.T) {
	tag := func(name string) mod.Tag { return mod.SkillName(name).Transfigured(true) }
	for _, tc := range []struct {
		name  string
		tag   mod.Tag
		cfg   *ListCfg
		match bool
	}{
		{"vaal-haste", tag("Haste"), &ListCfg{SkillName: "Haste", SkillGrantedEffect: gateGrantedEffect{id: "VaalHaste"}}, true},
		{"vaal-fireball", tag("Fireball"), &ListCfg{SkillName: "Fireball", SkillGrantedEffect: gateGrantedEffect{id: "VaalFireball"}}, true},
		{"transfigured", tag("Lightning Tendrils"), &ListCfg{SkillName: "Lightning Tendrils of Eccentricity", SkillGrantedEffect: gateGrantedEffect{id: "LightningTendrilsAltX"}}, true},
		{"plain", tag("Frenzy"), &ListCfg{SkillName: "Frenzy", SkillGrantedEffect: gateGrantedEffect{id: "Frenzy"}}, true},
		{"other-skill", tag("Haste"), &ListCfg{SkillName: "Frenzy", SkillGrantedEffect: gateGrantedEffect{id: "Frenzy"}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := NewModDB()
			db.Actor = gateActor{}
			db.AddMod(mod.NewFloat("AuraEffect", mod.TypeIncrease, 10).Tag(tc.tag))
			got := db.Sum(mod.TypeIncrease, tc.cfg, "AuraEffect") == 10
			if got != tc.match {
				t.Errorf("match = %v, want %v", got, tc.match)
			}
		})
	}
}
