package calculator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestModParserPinRowsMatchThePin: one ModCache line per template that the pin's ModParser.lua rows and its
// "within N metres" retry parse whole, with PoB's rendered mods.
//
//pob:src/Modules/ModParser.lua:7146-7161
func TestModParserPinRowsMatchThePin(t *testing.T) {
	requireGameData(t, "3.29")

	raw, err := os.ReadFile(filepath.Join("testdata", "modcache_pin_parser_rows.tsv"))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	n := 0
	for _, ln := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		line, want, ok := strings.Cut(ln, "\t")
		if !ok {
			t.Fatalf("REFUSING: malformed fixture row %q", ln)
		}
		n++
		e := ParseMod(line, false)
		var ours []mcdMod
		for _, m := range e.ModList {
			ours = append(ours, mcdOurMod(m))
		}
		if got := mcdField(ours, mcdMod.full); got != want || e.Extra != "" {
			t.Errorf("%q\n    pob:  %s\n    ours: %s (extra %q)", line, want, got, e.Extra)
		}
	}
	if n == 0 {
		t.Fatal("REFUSING: empty fixture")
	}
}
