package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/data"
)

// TestLegionStatLine: legionStatLine is the port of PassiveSpec.lua:1203-1220.
//
//	local replaceHelperFunc = function(statToFix, statKey, statMod, value)
//		if statMod.fmt == "g" then
//			if statKey:find("per_minute") then value = round(value / 60, 1)
//			elseif statKey:find("permyriad") then value = value / 100
//			elseif statKey:find("_ms") then value = value / 1000 end
//		end
//		if statMod.min ~= statMod.max then
//			return statToFix:gsub("%(" .. statMod.min .. "%-" .. statMod.max .. "%)", value)
//		elseif statMod.min ~= value then
//		end
//	end
func TestLegionStatLine(t *testing.T) {
	for _, c := range []struct {
		name     string
		template string
		stat     data.LegionStat
		value    float64
		want     string
	}{
		{
			name:     "range substitution",
			template: "(2-4)% increased maximum Life",
			stat:     data.LegionStat{Key: "maximum_life_+%", Fmt: "d", Index: 1, Min: 2, Max: 4},
			value:    2,
			want:     "2% increased maximum Life",
		},
		{
			name:     "range substitution, top of range",
			template: "(2-4)% increased maximum Life",
			stat:     data.LegionStat{Key: "maximum_life_+%", Fmt: "d", Index: 1, Min: 2, Max: 4},
			value:    4,
			want:     "4% increased maximum Life",
		},
		{
			name:     "fixed stat, combined value",
			template: "Minions deal 80% increased Damage",
			stat:     data.LegionStat{Key: "minion_damage_+%", Fmt: "d", Index: 1, Min: 80, Max: 80},
			value:    160,
			want:     "Minions deal 160% increased Damage",
		},
		{
			name:     "fixed stat, unchanged",
			template: "Minions deal 80% increased Damage",
			stat:     data.LegionStat{Key: "minion_damage_+%", Fmt: "d", Index: 1, Min: 80, Max: 80},
			value:    80,
			want:     "Minions deal 80% increased Damage",
		},
		{
			name:     "per_minute is not converted under fmt d",
			template: "(120-180) Life Regenerated per minute",
			stat:     data.LegionStat{Key: "life_regeneration_rate_per_minute", Fmt: "d", Index: 1, Min: 120, Max: 180},
			value:    120,
			want:     "120 Life Regenerated per minute",
		},
		{
			name:     "per_minute under fmt g",
			template: "Regenerate (2-3) Life per second",
			stat:     data.LegionStat{Key: "life_regeneration_rate_per_minute", Fmt: "g", Index: 1, Min: 2, Max: 3},
			value:    120,
			want:     "Regenerate 2 Life per second",
		},
		{
			name:     "permyriad under fmt g",
			template: "(1-2)% of Damage Leeched",
			stat:     data.LegionStat{Key: "life_leech_permyriad", Fmt: "g", Index: 1, Min: 1, Max: 2},
			value:    150,
			want:     "1.5% of Damage Leeched",
		},
		{
			name:     "_ms under fmt g",
			template: "(1-2) second duration",
			stat:     data.LegionStat{Key: "duration_ms", Fmt: "g", Index: 1, Min: 1, Max: 2},
			value:    1500,
			want:     "1.5 second duration",
		},
		{
			name:     "zero stat leaves the line alone",
			template: "Minions have 25% chance to Blind on Hit",
			stat:     data.LegionStat{},
			value:    0,
			want:     "Minions have 25% chance to Blind on Hit",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := legionStatLine(c.template, c.stat, c.value); got != c.want {
				t.Errorf("legionStatLine(%q, %+v, %g)\n got %q\nwant %q", c.template, c.stat, c.value, got, c.want)
			}
		})
	}
}
