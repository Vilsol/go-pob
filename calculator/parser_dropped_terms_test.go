package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestParserRowsCarryEveryModThePinEmits is a DROPPED-TERM guard.
func TestParserRowsCarryEveryModThePinEmits(t *testing.T) {
	requireGameData(t, "3.29")

	cases := []struct {
		line     string
		want     []string
		pin      string
		consumer string
		wantBase map[string]float64
	}{
		{
			line:     "Unholy Might",
			want:     []string{"Condition:UnholyMight", "Condition:CanWither"},
			pin:      "ModParser.lua:3976",
			consumer: "the wither config handler and calc_perform's wither application",
		},
		{
			line:     "You can inflict up to 5 Ignites on an Enemy",
			want:     []string{"IgniteCanStack", "IgniteStacks"},
			pin:      "ModParser.lua:3650",
			consumer: "calc_offence.go, Override(cfg, \"IgniteStacks\") in the ignite stack cap",
		},
		{
			line:     "Your Strength is added to your Minions",
			want:     []string{"StrengthAddedToMinions"},
			pin:      "ModParser.lua:4372 -- a BASE PERCENTAGE, not a flag",
			wantBase: map[string]float64{"StrengthAddedToMinions": 100},
			consumer: "calc_perform.go, CalcPerform.lua:1207-1209's " +
				"Sum(BASE, nil, \"StrengthAddedToMinions\")",
		},
		{
			line:     "Half of your Strength is added to your Minions",
			want:     []string{"StrengthAddedToMinions"},
			pin:      "ModParser.lua:4373 -- the SAME name at 50, not a second flag",
			wantBase: map[string]float64{"StrengthAddedToMinions": 50},
			consumer: "calc_perform.go, CalcPerform.lua:1207-1209's " +
				"Sum(BASE, nil, \"StrengthAddedToMinions\")",
		},
		{
			line:     "Modifiers to number of Projectiles instead apply to the number of targets Projectiles Split towards",
			want:     []string{"NoAdditionalProjectiles", "AdditionalProjectilesAddSplitsInstead"},
			pin:      "ModParser.lua:4572-4575",
			consumer: "calc_offence.go, the split-count branch",
		},
		{
			line:     "Leech applies instantly on Critical Strike",
			want:     []string{"InstantLifeLeech", "InstantManaLeech", "InstantEnergyShieldLeech"},
			pin:      "ModParser.lua:4600-4604",
			consumer: "calc_offence.go, EnergyShieldLeechInstantProportion",
		},
		{
			line:     "Gain Life and Mana from Leech instantly on Critical Strike",
			want:     []string{"InstantLifeLeech", "InstantManaLeech", "InstantEnergyShieldLeech"},
			pin:      "ModParser.lua:4605-4609",
			consumer: "calc_offence.go, EnergyShieldLeechInstantProportion",
		},
		{
			line:     "Leech applies instantly during Flask Effect",
			want:     []string{"InstantLifeLeech", "InstantManaLeech", "InstantEnergyShieldLeech"},
			pin:      "ModParser.lua:4610-4614",
			consumer: "calc_offence.go, EnergyShieldLeechInstantProportion",
		},

		{
			line:     "Gain Life and Mana from Leech instantly during Flask Effect",
			want:     []string{"InstantLifeLeech", "InstantManaLeech"},
			pin:      "ModParser.lua:4615-4618 -- upstream has NO energy-shield arm here",
			consumer: "n/a: this case asserts an absence",
		},
		{
			line:     "Life and Mana Leech are instant during Flask Effect",
			want:     []string{"InstantLifeLeech", "InstantManaLeech"},
			pin:      "ModParser.lua:4619-4622 -- upstream has NO energy-shield arm here",
			consumer: "n/a: this case asserts an absence",
		},
	}

	if c := ParseMod("10% increased Critical Strike Multiplier", false); c == nil || len(c.ModList) == 0 {
		t.Fatal("REFUSING: the control line parsed to nothing, so this table cannot distinguish " +
			"a dropped term from a broken call")
	}

	for _, tc := range cases {
		e := ParseMod(tc.line, false)
		if e == nil {
			t.Errorf("%s: ParseMod returned nothing for %q", tc.pin, tc.line)
			continue
		}
		if e.Extra != "" {
			t.Errorf("%s: ParseMod left extra %q on %q", tc.pin, e.Extra, tc.line)
		}

		got := map[string]bool{}
		for _, m := range e.ModList {
			got[m.Name()] = true
			want, pinned := tc.wantBase[m.Name()]
			if !pinned {
				continue
			}
			if m.Type() != mod.TypeBase {
				t.Errorf("%s: %q emits %s as %s, want BASE -- a flag standing in for a BASE "+
					"percentage carries the same NAME, so only the type separates them",
					tc.pin, tc.line, m.Name(), m.Type())
				continue
			}
			if v := m.Value().Float(); v != want {
				t.Errorf("%s: %q emits %s BASE %v, want %v", tc.pin, tc.line, m.Name(), v, want)
			}
		}
		for _, w := range tc.want {
			if !got[w] {
				t.Errorf("%s: %q produced no %s. Consumer: %s", tc.pin, tc.line, w, tc.consumer)
			}
		}
		if len(got) != len(tc.want) {
			names := make([]string, 0, len(got))
			for n := range got {
				names = append(names, n)
			}
			t.Errorf("%s: %q produced %v, want exactly %v -- an EXTRA mod is as much a "+
				"divergence as a missing one, and it is the direction no missing-thing "+
				"instrument looks", tc.pin, tc.line, names, tc.want)
		}
	}
}
