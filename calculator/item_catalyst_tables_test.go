package calculator

import "testing"

// TestCatalystTagsMatchThePin covers Item.lua:14-29 + :31-62.
func TestCatalystTagsMatchThePin(t *testing.T) {
	index := func(name string) *int {
		for i, n := range catalystList {
			if n == name {
				return intPtr(i + 1)
			}
		}
		return nil
	}
	quality := intPtr(20)

	for _, c := range []struct {
		catalyst string
		tag      string
		scales   bool
	}{
		{"Abrasive", "attack", true},
		{"Abrasive", "speed", false},
		{"Prismatic", "resistance", true},
		{"Tempering", "defences", true},

		{"Noxious", "physical_damage", true},
		{"Noxious", "chaos_damage", true},
		{"Noxious", "physical", false},
		{"Noxious", "chaos", false},

		{"Tempering", "armour", true},
		{"Tempering", "evasion", true},
		{"Tempering", "energyshield", true},

		{"Dextral", "suffix", true},
		{"Sinistral", "prefix", true},
	} {
		id := index(c.catalyst)
		if id == nil {
			t.Errorf("catalystList has no %q (the pin's list at Item.lua:14 has 12 entries)", c.catalyst)
			continue
		}
		tags := []string{c.tag}
		var prefix, suffix bool
		switch c.tag {
		case "prefix":
			tags, prefix = []string{"life"}, true
		case "suffix":
			tags, suffix = []string{"life"}, true
		}
		got := getCatalystScalar(id, tags, quality, prefix, suffix)
		want := 1.0
		if c.scales {
			want = 1.2
		}
		if got != want {
			t.Errorf("%s catalyst on a {tags:%s} line scaled by %v, want %v", c.catalyst, c.tag, got, want)
		}
	}
}

// TestCatalystLineFlagsSelectPrefixAndSuffix covers Item.lua:48-53.
func TestCatalystLineFlagsSelectPrefixAndSuffix(t *testing.T) {
	requireGameData(t, "3.29")

	const raw = "Rarity: RARE\nProbe Ring\nTwo-Stone Ring\n" +
		"Unique ID: 0000\nItem Level: 84\nQuality: 0\nLevelReq: 60\n" +
		"Catalyst: Sinistral\nCatalystQuality: 20\nImplicits: 0\n" +
		"{prefix}{tags:life}+100 to maximum Life\n"

	item := ParseItem(raw)
	if len(item.ExplicitModLines) != 1 {
		t.Fatalf("REFUSING: fixture produced %d explicit mod lines, want 1", len(item.ExplicitModLines))
	}
	if !item.ExplicitModLines[0].Prefix {
		t.Errorf("{prefix} did not reach the mod line, so a Sinistral catalyst can never match it")
	}
}

// TestCatalystQualitySpecSelectsTheCatalyst covers Item.lua:649-655.
func TestCatalystQualitySpecSelectsTheCatalyst(t *testing.T) {
	requireGameData(t, "3.29")

	const raw = "Rarity: RARE\nProbe Ring\nTwo-Stone Ring\n" +
		"Unique ID: 0000\nItem Level: 84\nQuality (Attribute Modifiers): +20%\n" +
		"LevelReq: 60\nImplicits: 0\n" +
		"{tags:attribute}+30 to Strength\n"

	item := ParseItem(raw)
	if item.Catalyst == nil || catalystList[*item.Catalyst-1] != "Intrinsic" {
		t.Errorf("catalyst = %v, want the Intrinsic index (%d)", item.Catalyst, 6)
	}
	if item.CatalystQuality == nil || *item.CatalystQuality != 20 {
		t.Errorf("catalystQuality = %v, want 20 (read from the digits before the %%, not the whole value)", item.CatalystQuality)
	}
	if leaked := leakedSpecLines(item); len(leaked) != 0 {
		t.Errorf("the spec still leaked %d lines into the mod lists: %q", len(leaked), leaked)
	}
}
