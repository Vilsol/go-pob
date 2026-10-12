package calculator

import (
	"testing"

	tjdata "github.com/Vilsol/timeless-jewels/data"
)

// TestConquerorListCoversEveryPoBConqueror pins the parser table against PoB's.
func TestConquerorListCoversEveryPoBConqueror(t *testing.T) {
	want := map[string]struct{ id, typ string }{
		"xibaqua": {"1", "vaal"}, "zerphi": {"2", "vaal"}, "doryani": {"3", "vaal"}, "ahuana": {"2_v2", "vaal"},
		"deshret": {"1", "maraketh"}, "asenath": {"2", "maraketh"}, "nasima": {"3", "maraketh"}, "balbala": {"1_v2", "maraketh"},
		"cadiro": {"1", "eternal"}, "victario": {"2", "eternal"}, "chitus": {"3", "eternal"}, "caspiro": {"3_v2", "eternal"},
		"kaom": {"1", "karui"}, "rakiata": {"2", "karui"}, "kiloava": {"3", "karui"}, "akoya": {"3_v2", "karui"},
		"venarius": {"1", "templar"}, "dominus": {"2", "templar"}, "avarius": {"3", "templar"}, "maxarius": {"1_v2", "templar"},
		"vorana": {"1", "kalguur"}, "uhtred": {"2", "kalguur"}, "medved": {"3", "kalguur"},
		"tecrod": {"1", "abyss_murderous"}, "ulaman": {"1", "abyss_searching"}, "kurgal": {"1", "abyss_hypnotic"},
		"amanamu": {"1", "abyss_ghastly"}, "zorath": {"1", "abyss_special"},
	}

	for name, w := range want {
		got, ok := conquerorList[name]
		if !ok {
			t.Errorf("conqueror %q is absent — it will parse to a zero ConquerorType and silently do nothing", name)
			continue
		}
		if got.ID != w.id || got.Type != w.typ {
			t.Errorf("conqueror %q = {ID:%q Type:%q}, want {ID:%q Type:%q}", name, got.ID, got.Type, w.id, w.typ)
		}
	}
	if len(conquerorList) != len(want) {
		t.Errorf("conquerorList has %d entries, PoB has %d", len(conquerorList), len(want))
	}
}

// TestJewelTypeByConquerorMatchesPoB pins the type table, and specifically that "kalguur" resolves.
func TestJewelTypeByConquerorMatchesPoB(t *testing.T) {
	for typ, want := range map[string]int{
		"vaal": 1, "karui": 2, "maraketh": 3, "templar": 4, "eternal": 5, "kalguur": 6,
		"abyss_murderous": 7, "abyss_searching": 8, "abyss_hypnotic": 9,
		"abyss_ghastly": 10, "abyss_special": 11,
	} {
		if got := legionJewelType(typ); got != want {
			t.Errorf("legionJewelType(%q) = %d, want %d", typ, got, want)
		}
	}

	if jt, ok := legionJewelTypeFor("kalguur"); !ok || jt != tjdata.HeroicTragedy {
		t.Errorf("legionJewelTypeFor(\"kalguur\") = %d,%v — want HeroicTragedy,true", jt, ok)
	}
	for _, abyss := range []string{"abyss_murderous", "abyss_searching", "abyss_hypnotic", "abyss_ghastly", "abyss_special"} {
		if _, ok := legionJewelTypeFor(abyss); ok {
			t.Errorf("legionJewelTypeFor(%q) accepted an Abyss type; the library cannot resolve it "+
				"and would return an empty struct that reads as 'this jewel does nothing'", abyss)
		}
	}
	if _, ok := legionJewelTypeFor("nonsense"); ok {
		t.Error("legionJewelTypeFor accepted an unknown conqueror type")
	}
}

// TestLegionWardLine pins the kalguur branch, PassiveSpec.lua:1360-1362.
func TestLegionWardLine(t *testing.T) {
	if got, want := legionWardLine(false), "2% increased Ward"; got != want {
		t.Errorf("legionWardLine(false) = %q, want %q", got, want)
	}
	if got, want := legionWardLine(true), "1% increased Ward"; got != want {
		t.Errorf("legionWardLine(true) = %q, want %q", got, want)
	}
}
