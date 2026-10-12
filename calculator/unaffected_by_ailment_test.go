package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

// TestUnaffectedByAilmentCoversEveryAilment pins ModParser.lua:4924-4941 (plus :3005, :3052-3053).
func TestUnaffectedByAilmentCoversEveryAilment(t *testing.T) {
	phrase := map[data.Ailment]string{data.AilmentBleed: "Bleeding"}
	for _, ailment := range data.Ailment("").Values() {
		word := string(ailment)
		if p, ok := phrase[ailment]; ok {
			word = p
		}
		line := "Unaffected by " + word
		entry := ParseMod(line, false)
		if strings.Trim(entry.Extra, " ") != "" {
			t.Errorf("%q: unparsed remainder %q -- the WHOLE line is discarded", line, entry.Extra)
			continue
		}
		if len(entry.ModList) != 1 {
			t.Errorf("%q: got %d mods, want 1", line, len(entry.ModList))
			continue
		}
		m := entry.ModList[0]
		want := "Self" + string(ailment) + "Effect"
		if m.Name() != want || m.Type() != mod.TypeMore {
			t.Errorf("%q: got %s %s, want %s MORE", line, m.Name(), m.Type(), want)
		}
	}
}

// TestUnaffectedByDamagingAilmentsEmitsAllThree pins ModParser.lua:4935-4939.
func TestUnaffectedByDamagingAilmentsEmitsAllThree(t *testing.T) {
	entry := ParseMod("Unaffected by Damaging Ailments", false)
	if strings.Trim(entry.Extra, " ") != "" {
		t.Fatalf("unparsed remainder %q", entry.Extra)
	}
	got := map[string]bool{}
	for _, m := range entry.ModList {
		got[m.Name()] = true
	}
	for _, want := range []string{"SelfBleedEffect", "SelfIgniteEffect", "SelfPoisonEffect"} {
		if !got[want] {
			t.Errorf("missing %s; got %v", want, got)
		}
	}
}
