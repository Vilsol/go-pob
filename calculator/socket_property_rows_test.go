package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestSocketPropertyRows pins ModParser.lua:3240-3242 and the `sockets` arm of the SocketedIn tag (ModStore.lua:741-748) they are the only producer for.
func TestSocketPropertyRows(t *testing.T) {
	for _, c := range []struct {
		line    string
		modName string
		keyword string
		value   float64
	}{
		{"+97% to Fire Resistance when Socketed with a Red Gem", "FireResist", "strength", 97},
		{"+100% to Cold Resistance when Socketed with a Green Gem", "ColdResist", "dexterity", 100},
		{"+99% to Lightning Resistance when Socketed with a Blue Gem", "LightningResist", "intelligence", 99},
	} {
		t.Run(c.modName, func(t *testing.T) {
			parsed := ParseMod(c.line, false)
			if parsed == nil || len(parsed.ModList) != 1 {
				t.Fatalf("ParseMod(%q) produced %v, want one mod", c.line, parsed)
			}
			m := parsed.ModList[0]
			if m.Name() != "SocketProperty" {
				t.Fatalf("mod name = %q, want SocketProperty", m.Name())
			}
			inner, ok := m.Value().ValueList.(mod.SocketProperty)
			if !ok {
				t.Fatalf("mod value is %T, want mod.SocketProperty", m.Value().ValueList)
			}
			if inner.Mod == nil || inner.Mod.Name() != c.modName {
				t.Fatalf("nested mod = %v, want %s", inner.Mod, c.modName)
			}

			var socketed *mod.SocketedInTag
			for _, tag := range m.Tags() {
				if s, isSocketed := tag.(*mod.SocketedInTag); isSocketed {
					socketed = s
				}
			}
			if socketed == nil {
				t.Fatalf("no SocketedIn tag on %q", c.line)
			}
			if socketed.TagKeyword != c.keyword {
				t.Fatalf("tag keyword = %q, want %q — a red gem is an ATTRIBUTE, not a socket colour", socketed.TagKeyword, c.keyword)
			}
			if len(socketed.TagSockets) != 1 || socketed.TagSockets[0] != 1 {
				t.Fatalf("tag sockets = %v, want [1]", socketed.TagSockets)
			}

			db := moddb.NewModDB()
			db.AddMod(m)
			gem := &poe.SkillGem{}
			old := moddb.GemIsType
			moddb.GemIsType = func(*poe.SkillGem, string) bool { return true }
			defer func() { moddb.GemIsType = old }()

			for _, sc := range []struct {
				name      string
				socketNum int
				want      int
			}{
				{"socket 1", 1, 1},
				{"socket 2", 2, 0},
				{"no socket named", 0, 0},
			} {
				got := db.List(&moddb.ListCfg{SlotName: socketed.SlotName, SkillGem: gem, SocketNum: sc.socketNum}, "SocketProperty")
				if len(got) != sc.want {
					t.Fatalf("%s: List returned %d mods, want %d", sc.name, len(got), sc.want)
				}
			}
		})
	}
}
