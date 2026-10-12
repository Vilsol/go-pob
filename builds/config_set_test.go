package builds

import "testing"

// TestConfigSetInputsAreRead pins ConfigTab.lua:837-870's Load.
func TestConfigSetInputsAreRead(t *testing.T) {
	const header = `<?xml version="1.0" encoding="UTF-8"?><PathOfBuilding>` +
		`<Build level="90" className="Witch" ascendClassName="Occultist"/>` +
		`<Tree activeSpec="1"><Spec treeVersion="3_29" nodes=""/></Tree>`

	for _, tc := range []struct {
		name, xml, want string
	}{
		{
			name: "flat Config, no sets",
			xml: header + `<Config>
				<Input name="enemyIsBoss" string="None"/>
			</Config></PathOfBuilding>`,
			want: "None",
		},
		{
			name: "single set, active 1",
			xml: header + `<Config activeConfigSet="1">
				<ConfigSet id="1"><Input name="enemyIsBoss" string="Boss"/></ConfigSet>
			</Config></PathOfBuilding>`,
			want: "Boss",
		},
		{
			name: "two sets, active 2 — must NOT take the first",
			xml: header + `<Config activeConfigSet="2">
				<ConfigSet id="1"><Input name="enemyIsBoss" string="None"/></ConfigSet>
				<ConfigSet id="2"><Input name="enemyIsBoss" string="Uber"/></ConfigSet>
			</Config></PathOfBuilding>`,
			want: "Uber",
		},
		{
			name: "sets in descending id order, active 1",
			xml: header + `<Config activeConfigSet="1">
				<ConfigSet id="2"><Input name="enemyIsBoss" string="Uber"/></ConfigSet>
				<ConfigSet id="1"><Input name="enemyIsBoss" string="Pinnacle"/></ConfigSet>
			</Config></PathOfBuilding>`,
			want: "Pinnacle",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			build, err := ParseBuildStr(tc.xml)
			if err != nil {
				t.Fatalf("ParseBuild: %v", err)
			}
			if got := build.GetStringOption("enemyIsBoss"); got != tc.want {
				t.Errorf("enemyIsBoss = %q, want %q — %d input(s) resolved",
					got, tc.want, len(build.Config.Inputs))
			}
		})
	}
}

// TestConfigSetResolutionMatchesPoB: What the pinned PoB holds after loading 0280 with its <Config> rewritten.
func TestConfigSetResolutionMatchesPoB(t *testing.T) {
	const header = `<?xml version="1.0" encoding="UTF-8"?><PathOfBuilding>` +
		`<Build level="90" className="Witch" ascendClassName="Occultist"/>` +
		`<Tree activeSpec="1"><Spec treeVersion="3_29" nodes=""/></Tree>`
	set1 := `<ConfigSet id="1" title="Default"><Input name="ActiveBrands" number="2"/></ConfigSet>`
	set2 := `<ConfigSet id="2" title="Two"><Input name="ActiveBrands" number="5"/></ConfigSet>`
	loose := `<Input name="ActiveBrands" number="3"/>`

	setValue := func(t *testing.T, saved []byte, id int) float64 {
		t.Helper()
		back, err := ParseBuild(saved)
		if err != nil {
			t.Fatalf("ParseBuild (saved): %v", err)
		}
		for _, s := range back.Config.ConfigSets {
			if s.ID == id {
				for _, in := range s.Inputs {
					if in.Name == "ActiveBrands" && in.Number != nil {
						return *in.Number
					}
				}
			}
		}
		t.Fatalf("the saved build has no ActiveBrands in set %d", id)
		return 0
	}

	for _, tc := range []struct {
		name, config string
		active       int
		live         float64
		sets         map[int]float64
	}{
		{"A: active id names no set, set 1 not first", `<Config activeConfigSet="7">` + set2 + set1 + `</Config>`, 2, 5, map[int]float64{1: 2, 2: 5}},
		{"B: a loose input after the sets, set 2 active", `<Config activeConfigSet="2">` + set1 + set2 + loose + `</Config>`, 2, 5, map[int]float64{1: 3, 2: 5}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			build, err := ParseBuildStr(header + tc.config + `</PathOfBuilding>`)
			if err != nil {
				t.Fatalf("ParseBuild: %v", err)
			}
			if got := build.GetNumberOption("ActiveBrands"); got != tc.live {
				t.Errorf("live ActiveBrands = %v, want %v", got, tc.live)
			}
			if got := build.Config.ConfigSets[build.ActiveConfigSetIndex()].ID; got != tc.active {
				t.Errorf("active set = %d, want %d", got, tc.active)
			}
			saved, err := SerializeBuild(build)
			if err != nil {
				t.Fatalf("SerializeBuild: %v", err)
			}
			for id, want := range tc.sets {
				if got := setValue(t, saved, id); got != want {
					t.Errorf("saved set %d ActiveBrands = %v, want %v", id, got, want)
				}
			}
		})
	}
}
