package builds

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

// TestCustomModifierBlockRoundTrips pins ConfigTab.lua:842-864 (Load) against :941-953 (Save).
func TestCustomModifierBlockRoundTrips(t *testing.T) {
	const header = `<?xml version="1.0" encoding="UTF-8"?><PathOfBuilding>` +
		`<Build level="90" className="Witch" ascendClassName="Occultist"/>` +
		`<Tree activeSpec="1"><Spec treeVersion="3_29" nodes=""/></Tree>`

	in := header + `<Config activeConfigSet="1">
		<ConfigSet id="1">
			<Input name="enemyIsBoss" string="Boss"/>
			<CustomModifierBlock title="Default" enabled="true">10% increased Strength
20% increased Dexterity</CustomModifierBlock>
			<CustomModifierBlock title="Group 2" enabled="false">30% increased Intelligence</CustomModifierBlock>
		</ConfigSet>
	</Config></PathOfBuilding>`

	build, err := ParseBuildStr(in)
	if err != nil {
		t.Fatalf("ParseBuild: %v", err)
	}

	blocks := build.Config.CustomModifierBlocks
	if len(blocks) != 2 {
		t.Fatalf("resolved %d block(s), want 2 — the active <ConfigSet>'s blocks are not reaching Config", len(blocks))
	}
	if got := blocks[0].BlockTitle(); got != "Default" {
		t.Errorf("block 0 title = %q, want %q", got, "Default")
	}
	if !blocks[0].IsEnabled() {
		t.Error(`block 0 enabled="true" read as disabled`)
	}
	if blocks[1].IsEnabled() {
		t.Error(`block 1 enabled="false" read as enabled — ConfigTab.lua:845 is an exact "true" compare`)
	}
	if !strings.Contains(blocks[0].Text, "10% increased Strength") ||
		!strings.Contains(blocks[0].Text, "20% increased Dexterity") {
		t.Errorf("block 0 text = %q, want both mod lines", blocks[0].Text)
	}
	if !strings.Contains(blocks[1].Text, "30% increased Intelligence") {
		t.Errorf("block 1 text = %q, want the disabled block's line", blocks[1].Text)
	}

	out, err := SerializeBuild(build)
	if err != nil {
		t.Fatalf("SerializeBuild: %v", err)
	}
	saved := string(out)
	for _, want := range []string{
		`title="Default"`,
		`title="Group 2"`,
		`enabled="false"`,
		"10% increased Strength",
		"20% increased Dexterity",
		"30% increased Intelligence",
	} {
		if !strings.Contains(saved, want) {
			t.Errorf("saved XML is missing %q", want)
		}
	}
	if n := strings.Count(saved, "<CustomModifierBlock"); n != 2 {
		t.Errorf("saved %d <CustomModifierBlock> element(s), want 2", n)
	}

	again, err := ParseBuildStr(saved)
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}
	if len(again.Config.CustomModifierBlocks) != 2 {
		t.Fatalf("reload resolved %d block(s), want 2 — the save is not a fixed point", len(again.Config.CustomModifierBlocks))
	}
	if again.Config.CustomModifierBlocks[1].IsEnabled() {
		t.Error("block 1 came back enabled after a save/load cycle")
	}
}

// TestCustomModifierBlockAttributeDefaults pins the two absent-attribute rules.
func TestCustomModifierBlockAttributeDefaults(t *testing.T) {
	const header = `<?xml version="1.0" encoding="UTF-8"?><PathOfBuilding>` +
		`<Build level="90" className="Witch" ascendClassName="Occultist"/>` +
		`<Tree activeSpec="1"><Spec treeVersion="3_29" nodes=""/></Tree>`

	for _, tc := range []struct {
		name, attrs string
		wantTitle   string
		wantEnabled bool
	}{
		{name: "both absent", attrs: ``, wantTitle: "Default", wantEnabled: true},
		{name: "enabled true", attrs: ` enabled="true"`, wantTitle: "Default", wantEnabled: true},
		{name: "enabled false", attrs: ` enabled="false"`, wantTitle: "Default", wantEnabled: false},
		{name: "enabled TRUE is not true", attrs: ` enabled="TRUE"`, wantTitle: "Default", wantEnabled: false},
		{name: "enabled garbage is false, not an error", attrs: ` enabled="yes"`, wantTitle: "Default", wantEnabled: false},
		{name: "empty title is not the default", attrs: ` title=""`, wantTitle: "", wantEnabled: true},
		{name: "titled", attrs: ` title="Nimis"`, wantTitle: "Nimis", wantEnabled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			build, err := ParseBuildStr(header + `<Config activeConfigSet="1"><ConfigSet id="1">` +
				`<CustomModifierBlock` + tc.attrs + `>x</CustomModifierBlock>` +
				`</ConfigSet></Config></PathOfBuilding>`)
			if err != nil {
				t.Fatalf("ParseBuild: %v", err)
			}
			if len(build.Config.CustomModifierBlocks) != 1 {
				t.Fatalf("resolved %d block(s), want 1", len(build.Config.CustomModifierBlocks))
			}
			b := build.Config.CustomModifierBlocks[0]
			if got := b.BlockTitle(); got != tc.wantTitle {
				t.Errorf("BlockTitle() = %q, want %q", got, tc.wantTitle)
			}
			if got := b.IsEnabled(); got != tc.wantEnabled {
				t.Errorf("IsEnabled() = %v, want %v", got, tc.wantEnabled)
			}
		})
	}
}

// TestCustomModifierBlockEditsReachTheSave is the half the round-trip test above CANNOT see.
func TestCustomModifierBlockEditsReachTheSave(t *testing.T) {
	const header = `<?xml version="1.0" encoding="UTF-8"?><PathOfBuilding>` +
		`<Build level="90" className="Witch" ascendClassName="Occultist"/>` +
		`<Tree activeSpec="1"><Spec treeVersion="3_29" nodes=""/></Tree>`

	build, err := ParseBuildStr(header + `<Config activeConfigSet="1"><ConfigSet id="1">` +
		`<CustomModifierBlock title="Default" enabled="true">10% increased Strength</CustomModifierBlock>` +
		`</ConfigSet></Config></PathOfBuilding>`)
	if err != nil {
		t.Fatalf("ParseBuild: %v", err)
	}

	added := "Group 2"
	build.Config.CustomModifierBlocks = append(build.Config.CustomModifierBlocks, pob.CustomModifierBlock{
		Title: &added,
		Text:  "30% increased Intelligence",
	})
	build.Config.CustomModifierBlocks[0].Text = "50% increased Strength"

	out, err := SerializeBuild(build)
	if err != nil {
		t.Fatalf("SerializeBuild: %v", err)
	}
	saved := string(out)
	if n := strings.Count(saved, "<CustomModifierBlock"); n != 2 {
		t.Errorf("saved %d block(s), want 2 — the appended block never reached the <ConfigSet>", n)
	}
	for _, want := range []string{"30% increased Intelligence", "50% increased Strength"} {
		if !strings.Contains(saved, want) {
			t.Errorf("saved XML is missing %q", want)
		}
	}
	if strings.Contains(saved, "10% increased Strength") {
		t.Error("saved XML still carries the pre-edit text — the ConfigSet's stale copy was written")
	}
}
