package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/mod"
)

// TestEvalSocketedInTag covers PoB ModStore.lua:502-505.
//
//	if not cfg or tag.slotName ~= cfg.slotName or (tag.keyword and (not cfg or not cfg.skillGem
//	   or not calcLib.gemIsType(cfg.skillGem, tag.keyword))) then return end
func TestEvalSocketedInTag(t *testing.T) {
	bowGem := &poe.SkillGem{}
	swordGem := &poe.SkillGem{}

	prev := GemIsType
	GemIsType = func(gem *poe.SkillGem, keyword string) bool {
		return gem == bowGem && keyword == "bow"
	}
	t.Cleanup(func() { GemIsType = prev })

	tests := []struct {
		name string
		tag  mod.Tag
		cfg  *ListCfg
		want float64
	}{
		{
			name: "matching slot",
			tag:  mod.SocketedIn("Helmet"),
			cfg:  &ListCfg{SlotName: "Helmet"},
			want: 5,
		},
		{
			name: "different slot",
			tag:  mod.SocketedIn("Helmet"),
			cfg:  &ListCfg{SlotName: "Weapon 2"},
			want: 0,
		},
		{
			name: "empty slot on cfg",
			tag:  mod.SocketedIn("Helmet"),
			cfg:  &ListCfg{},
			want: 0,
		},
		{
			name: "nil cfg",
			tag:  mod.SocketedIn("Helmet"),
			cfg:  nil,
			want: 0,
		},
		{
			name: "unsubstituted placeholder never matches a real slot",
			tag:  mod.SocketedIn("{SlotName}"),
			cfg:  &ListCfg{SlotName: "Helmet"},
			want: 0,
		},
		{
			name: "keyword matches gem",
			tag:  mod.SocketedIn("Helmet").Keyword("bow"),
			cfg:  &ListCfg{SlotName: "Helmet", SkillGem: bowGem},
			want: 5,
		},
		{
			name: "keyword does not match gem",
			tag:  mod.SocketedIn("Helmet").Keyword("bow"),
			cfg:  &ListCfg{SlotName: "Helmet", SkillGem: swordGem},
			want: 0,
		},
		{
			name: "keyword with no gem",
			tag:  mod.SocketedIn("Helmet").Keyword("bow"),
			cfg:  &ListCfg{SlotName: "Helmet"},
			want: 0,
		},
		{
			name: "keyword matches but slot does not",
			tag:  mod.SocketedIn("Helmet").Keyword("bow"),
			cfg:  &ListCfg{SlotName: "Gloves", SkillGem: bowGem},
			want: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := NewModDB()
			db.Actor = gateActor{}
			db.AddMod(mod.NewFloat("test", mod.TypeBase, 5).Tag(test.tag))
			testza.AssertEqual(t, test.want, db.Sum(mod.TypeBase, test.cfg, "test"))
		})
	}
}

// TestEvalSocketedInTagWithoutGemIsType pins the fail-closed behaviour of the gemIsType seam.
func TestEvalSocketedInTagWithoutGemIsType(t *testing.T) {
	prev := GemIsType
	GemIsType = nil
	t.Cleanup(func() { GemIsType = prev })

	db := NewModDB()
	db.Actor = gateActor{}
	db.AddMod(mod.NewFloat("test", mod.TypeBase, 5).Tag(mod.SocketedIn("Helmet").Keyword("bow")))

	testza.AssertEqual(t, 0.0, db.Sum(mod.TypeBase, &ListCfg{SlotName: "Helmet", SkillGem: &poe.SkillGem{}}, "test"))
}
