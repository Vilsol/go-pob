package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestFlagRespectsValue: A FLAG mod carries a boolean value.
func TestFlagRespectsValue(t *testing.T) {
	tests := []struct {
		name string
		mods []mod.Mod
		cfg  *ListCfg
		want bool
	}{
		{
			name: "untagged true",
			mods: []mod.Mod{mod.NewFlag("testMod", true)},
			want: true,
		},
		{
			name: "untagged false",
			mods: []mod.Mod{mod.NewFlag("testMod", false)},
			want: false,
		},
		{
			name: "tagged false, gate passes",
			mods: []mod.Mod{mod.NewFlag("testMod", false).Tag(mod.SkillName("Fireball"))},
			cfg:  &ListCfg{SkillName: "Fireball"},
			want: false,
		},
		{
			name: "tagged true, gate passes",
			mods: []mod.Mod{mod.NewFlag("testMod", true).Tag(mod.SkillName("Fireball"))},
			cfg:  &ListCfg{SkillName: "Fireball"},
			want: true,
		},
		{
			name: "tagged true, gate fails",
			mods: []mod.Mod{mod.NewFlag("testMod", true).Tag(mod.SkillName("Fireball"))},
			cfg:  &ListCfg{SkillName: "Icebolt"},
			want: false,
		},
		{
			name: "tagged false, gate fails",
			mods: []mod.Mod{mod.NewFlag("testMod", false).Tag(mod.SkillName("Fireball"))},
			cfg:  &ListCfg{SkillName: "Icebolt"},
			want: false,
		},
		{
			name: "false then true keeps scanning",
			mods: []mod.Mod{mod.NewFlag("testMod", false), mod.NewFlag("testMod", true)},
			want: true,
		},
		{
			name: "true then false still true",
			mods: []mod.Mod{mod.NewFlag("testMod", true), mod.NewFlag("testMod", false)},
			want: true,
		},
		{
			name: "all false",
			mods: []mod.Mod{mod.NewFlag("testMod", false), mod.NewFlag("testMod", false)},
			want: false,
		},
	}

	for _, test := range tests {
		t.Run("ModList/"+test.name, func(t *testing.T) {
			m := NewModList()
			m.Actor = gateActor{}
			for _, mo := range test.mods {
				m.AddMod(mo)
			}
			testza.AssertEqual(t, test.want, m.Flag(test.cfg, "testMod"))
		})

		t.Run("ModDB/"+test.name, func(t *testing.T) {
			m := NewModDB()
			m.Actor = gateActor{}
			for _, mo := range test.mods {
				m.AddMod(mo)
			}
			testza.AssertEqual(t, test.want, m.Flag(test.cfg, "testMod"))
		})
	}
}

// TestFlagRespectsValueThroughParent: A false flag in a child store must not mask a true flag in the parent, and a false flag in the parent must not read as true.
func TestFlagRespectsValueThroughParent(t *testing.T) {
	t.Run("false child, true parent", func(t *testing.T) {
		parent := NewModDB()
		parent.Actor = gateActor{}
		parent.AddMod(mod.NewFlag("testMod", true))

		child := NewModDB()
		child.Actor = gateActor{}
		child.Parent = parent
		child.AddMod(mod.NewFlag("testMod", false))

		testza.AssertTrue(t, child.Flag(nil, "testMod"))
	})

	t.Run("false parent", func(t *testing.T) {
		parent := NewModDB()
		parent.Actor = gateActor{}
		parent.AddMod(mod.NewFlag("testMod", false))

		child := NewModDB()
		child.Actor = gateActor{}
		child.Parent = parent

		testza.AssertFalse(t, child.Flag(nil, "testMod"))
	})
}

// TestCombineFlagRespectsValue: Combine(TypeFlag) routes through Flag, so it must agree.
func TestCombineFlagRespectsValue(t *testing.T) {
	m := NewModList()
	m.Actor = gateActor{}
	m.AddMod(mod.NewFlag("testMod", false))

	testza.AssertFalse(t, m.Combine(mod.TypeFlag, nil, "testMod").Flag())
}
