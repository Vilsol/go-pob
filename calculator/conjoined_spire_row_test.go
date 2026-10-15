package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestConjoinedSpireTotemLifeRow pins ModParser.lua:3069-3071 (upstream #10286): the line scales totem life, not the player's.
func TestConjoinedSpireTotemLifeRow(t *testing.T) {
	entry := ParseMod("Totems have 5% more Life per maximum number of Summoned Totems", false)
	if strings.TrimSpace(entry.Extra) != "" || len(entry.ModList) != 1 {
		t.Fatalf("extra %q, %d mods; want one mod", entry.Extra, len(entry.ModList))
	}
	m := entry.ModList[0]
	if m.Name() != "TotemLife" || m.Type() != mod.TypeMore || m.Value().Float() != 5 || m.KeywordFlags() != 0 {
		t.Errorf("got %s %s %v kw=%#x, want TotemLife MORE 5 kw=0", m.Name(), m.Type(), m.Value().Float(), m.KeywordFlags())
	}
}
