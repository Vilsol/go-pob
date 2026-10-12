package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

const customBlockBuildHeader = `<?xml version="1.0" encoding="UTF-8"?><PathOfBuilding>` +
	`<Build level="90" className="Witch" ascendClassName="Occultist"/>` +
	`<Tree activeSpec="1"><Spec treeVersion="3_29" nodes=""/></Tree>`

func customBlockAuras(t *testing.T, configXML string) []string {
	t.Helper()

	build, err := builds.ParseBuildStr(customBlockBuildHeader + configXML + `</PathOfBuilding>`)
	if err != nil {
		t.Fatalf("ParseBuild: %v", err)
	}
	list, _ := buildConfigModLists(&Environment{Build: build}, build)

	var sources []string
	for _, m := range list.Mods() {
		if m.Name() == "ExtraAura" {
			sources = append(sources, string(m.GetSource()))
		}
	}
	return sources
}

// TestCustomModifierBlocksReachTheModList pins ConfigTab.lua:1106-1129.
func TestCustomModifierBlocksReachTheModList(t *testing.T) {
	requireGameData(t, "3.29")

	const line = "You and nearby allies gain 328 additional Energy Shield"

	t.Run("control: legacy customMods input still applies", func(t *testing.T) {
		got := customBlockAuras(t, `<Config activeConfigSet="1"><ConfigSet id="1">`+
			`<Input name="customMods" string="`+line+`"/></ConfigSet></Config>`)
		if len(got) != 1 {
			t.Fatalf("CONTROL FAILED: legacy customMods produced %d ExtraAura mods, want 1", len(got))
		}
	})

	t.Run("an enabled block applies, sourced by its title", func(t *testing.T) {
		got := customBlockAuras(t, `<Config activeConfigSet="1"><ConfigSet id="1">`+
			`<CustomModifierBlock title="Nimis" enabled="true">`+line+`</CustomModifierBlock>`+
			`</ConfigSet></Config>`)
		if len(got) != 1 {
			t.Fatalf("a non-empty enabled block produced %d ExtraAura mods, want 1", len(got))
		}
		if got[0] != "Custom:Nimis" {
			t.Errorf("mod source = %q, want %q", got[0], "Custom:Nimis")
		}
	})

	t.Run("a disabled block does not apply", func(t *testing.T) {
		got := customBlockAuras(t, `<Config activeConfigSet="1"><ConfigSet id="1">`+
			`<CustomModifierBlock title="Off" enabled="false">`+line+`</CustomModifierBlock>`+
			`</ConfigSet></Config>`)
		if len(got) != 0 {
			t.Errorf("a disabled block produced %d ExtraAura mods, want 0", len(got))
		}
	})

	t.Run("every enabled block applies, not just the first", func(t *testing.T) {
		got := customBlockAuras(t, `<Config activeConfigSet="1"><ConfigSet id="1">`+
			`<CustomModifierBlock title="A" enabled="true">`+line+`</CustomModifierBlock>`+
			`<CustomModifierBlock title="B" enabled="true">`+line+`</CustomModifierBlock>`+
			`</ConfigSet></Config>`)
		if len(got) != 2 {
			t.Errorf("two enabled blocks produced %d ExtraAura mods, want 2", len(got))
		}
	})

	t.Run("blocks come from the ACTIVE config set", func(t *testing.T) {
		got := customBlockAuras(t, `<Config activeConfigSet="2">`+
			`<ConfigSet id="1"><CustomModifierBlock title="A" enabled="true">`+line+`</CustomModifierBlock></ConfigSet>`+
			`<ConfigSet id="2"/>`+
			`</Config>`)
		if len(got) != 0 {
			t.Errorf("set 1's block applied while set 2 was active: %d ExtraAura mods, want 0", len(got))
		}
	})
}

// TestLegacyCustomModsMigrationPredicate pins ConfigTab.lua:876-877: the legacy text becomes a block titled "Default".
func TestLegacyCustomModsMigrationPredicate(t *testing.T) {
	requireGameData(t, "3.29")

	const line = "You and nearby allies gain 328 additional Energy Shield"
	legacy := `<Input name="customMods" string="` + line + `"/>`

	for _, tc := range []struct {
		name, blocks string
	}{
		{"no blocks", ``},
		{"one empty block", `<CustomModifierBlock title="Default" enabled="true"></CustomModifierBlock>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := customBlockAuras(t, `<Config activeConfigSet="1"><ConfigSet id="1">`+
				legacy+tc.blocks+`</ConfigSet></Config>`)
			if len(got) != 1 || got[0] != "Custom:Default" {
				t.Errorf("sources = %v, want exactly [Custom:Default]", got)
			}
		})
	}
}

// TestLegacyAndBlockCoexistence: ConfigTab.lua:876 migrates the legacy text only over no block or one empty block, and :882 drops it either way.
func TestLegacyAndBlockCoexistence(t *testing.T) {
	requireGameData(t, "3.29")

	const line = "You and nearby allies gain 328 additional Energy Shield"
	legacy := `<Input name="customMods" string="` + line + `"/>`

	for _, tc := range []struct {
		name, blocks string
		want         []string
		pob          string
	}{
		{
			name:   "two empty blocks beside legacy text",
			blocks: `<CustomModifierBlock title="A" enabled="true"></CustomModifierBlock><CustomModifierBlock title="B" enabled="true"></CustomModifierBlock>`,
			want:   nil,
			pob:    "no mods at all: #customModsList == 2 stops the migration and :882 nils the input",
		},
		{
			name:   "a text-carrying block beside legacy text",
			blocks: `<CustomModifierBlock title="A" enabled="true">` + line + `</CustomModifierBlock>`,
			want:   []string{"Custom:A"},
			pob:    "only the block's mod, sourced Custom:A",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := customBlockAuras(t, `<Config activeConfigSet="1"><ConfigSet id="1">`+
				legacy+tc.blocks+`</ConfigSet></Config>`)
			if len(got) != len(tc.want) {
				t.Fatalf("sources = %v, want %v (upstream would give %s)", got, tc.want, tc.pob)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("source[%d] = %q, want %q (upstream would give %s)", i, got[i], tc.want[i], tc.pob)
				}
			}
		})
	}
}

// TestCustomModifierLinesStripEscapes: ConfigTab.lua:1114 and :1133 strip colour escapes before parsing a custom line.
func TestCustomModifierLinesStripEscapes(t *testing.T) {
	requireGameData(t, "3.29")

	const line = "^7You and nearby allies gain ^x8888FF328^7 additional Energy Shield"
	for name, configXML := range map[string]string{
		"block":  `<Config activeConfigSet="1"><ConfigSet id="1"><CustomModifierBlock title="A" enabled="true">` + line + `</CustomModifierBlock></ConfigSet></Config>`,
		"legacy": `<Config activeConfigSet="1"><ConfigSet id="1"><Input name="customMods" string="` + line + `"/></ConfigSet></Config>`,
	} {
		if got := customBlockAuras(t, configXML); len(got) != 1 {
			t.Errorf("%s: an escaped line produced %d ExtraAura mods, want 1", name, len(got))
		}
	}
}
