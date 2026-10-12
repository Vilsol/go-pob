package calculator

import (
	"math"
	"testing"
)

// TestSortValuesEqualTheFullWhatIf: The popups' sort values skip the Full DPS roll-up for every other stat.
func TestSortValuesEqualTheFullWhatIf(t *testing.T) {
	initItemsTabData(t)
	for _, file := range []string{"0280-elementalist-fireball.xml", "0339-reliquarian-shield-crush-of-the-chieftain-2.xml"} {
		build := loadCorpusBuild(t, file)
		misc := GetMiscCalculator(build)
		_, helmet := activeSetSlot(build, "Helmet")
		item := ParseItem(helmet)
		var ids []string
		for _, g := range buildImplicitMods(item, "EATER") {
			for k, m := range g.Mods {
				if k%9 == 0 {
					ids = append(ids, m.ID)
				}
			}
		}
		if len(ids) < 10 {
			t.Fatalf("REFUSING: %d Eater implicits sampled for %s's helmet", len(ids), file)
		}
		texts, found := implicitSortTexts(helmet, item, "EATER", ids)
		for k, id := range ids {
			if text, ok := ImplicitText(helmet, "EATER", id, ""); text != texts[k] || ok != found[k] {
				t.Fatalf("%s %s: the item text differs from ImplicitText's", file, id)
			}
		}
		texts, found = append(texts, texts[0]), append(found, true)

		envs := make([]*Environment, len(texts))
		for k, text := range texts {
			envs[k] = misc.Calc(&CalcOverride{RepSlotName: item.getPrimarySlot(), RepItem: ParseItem(text)})
		}
		compared, nonzero := 0, 0
		for _, ps := range PowerStats() {
			got := misc.sortValues(item.getPrimarySlot(), texts, found, ps)
			for k := range texts {
				want := powerStatValue(envs[k], ps)
				if math.Float64bits(got[k]) != math.Float64bits(want) && !(math.IsNaN(got[k]) && math.IsNaN(want)) {
					t.Errorf("%s %s with %s: sort value %v, the full what-if %v", file, ps.Stat, ids[min(k, len(ids)-1)], got[k], want)
				}
				compared++
				if want != 0 {
					nonzero++
				}
			}
		}
		if nonzero < compared/4 {
			t.Fatalf("REFUSING: %s: only %d of %d values nonzero", file, nonzero, compared)
		}
	}
}

// TestSortFunctionsEqualTheRolledUpWhatIf: Every popup's public sort function equals itself with the Full DPS roll-up forced for every stat, bit for bit.
func TestSortFunctionsEqualTheRolledUpWhatIf(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, "0280-elementalist-fireball.xml")
	misc := GetMiscCalculator(build)
	_, helmet := activeSetSlot(build, "Helmet")
	_, amulet := activeSetSlot(build, "Amulet")
	item := ParseItem(helmet)
	every := func(ids []string, step int) []string {
		var out []string
		for k := 0; k < len(ids); k += step {
			out = append(out, ids[k])
		}
		return out
	}
	var eater, corrupt []string
	for _, g := range buildImplicitMods(item, "EATER") {
		for _, m := range g.Mods {
			eater = append(eater, m.ID)
		}
	}
	customMods := buildCustomMods(helmet, item, "PREFIX")
	prefix := make([]string, 0, len(customMods))
	for _, m := range customMods {
		prefix = append(prefix, m.ID)
	}
	for _, m := range itemAffixes(item) {
		if m.Type == "Corrupted" {
			corrupt = append(corrupt, m.ID)
		}
	}
	var notables []int64
	for k, n := range AnointNotables() {
		if k%60 == 0 {
			notables = append(notables, n.ID)
		}
	}
	circlet, _ := CraftItemText(2, "", "Hubris Circlet")
	runs := map[string]func(stat string) []float64{
		"implicit": func(s string) []float64 { return misc.ImplicitSortValues(helmet, "EATER", every(eater, 25), s) },
		"custom":   func(s string) []float64 { return misc.CustomModSortValues(helmet, "PREFIX", every(prefix, 6), s) },
		"corrupt": func(s string) []float64 {
			return misc.CorruptSortValues(helmet, "Corrupted", "Corrupted", every(corrupt, 25), s)
		},
		"enchant": func(s string) []float64 {
			return misc.EnchantSortValues(helmet, 1, []string{"+50 to maximum Life", "1% increased Fireball Cast Speed", "+50 to maximum Life"}, s)
		},
		"anoint": func(s string) []float64 { return misc.AnointPowers(amulet, 1, notables, s) },
		"affix":  func(s string) []float64 { return misc.AffixPowers(circlet, 1, s) },
	}
	for name, run := range runs {
		nonzero := 0
		for _, stat := range []string{"Life", "TotalEHP", "CombinedDPS", "ManaRegen"} {
			got := run(stat)
			sortValuesRollUpAll = true
			want := run(stat)
			sortValuesRollUpAll = false
			if len(got) < 3 || len(got) != len(want) {
				t.Fatalf("REFUSING: %s %s: %d values against %d", name, stat, len(got), len(want))
			}
			for k := range got {
				if math.Float64bits(got[k]) != math.Float64bits(want[k]) && !(math.IsNaN(got[k]) && math.IsNaN(want[k])) {
					t.Errorf("%s %s [%d]: %v, rolled up %v", name, stat, k, got[k], want[k])
				}
				if want[k] != 0 {
					nonzero++
				}
			}
		}
		if nonzero == 0 {
			t.Errorf("REFUSING: %s: every value is 0", name)
		}
	}
}
