package calculator

import (
	"os"
	"strings"
	"testing"
)

// TestFirePenUncappedFireResNameAgreement pins the producer and the consumer of `FirePenIncreasedByUncappedFireRes` to the SAME string literal.
func TestFirePenUncappedFireResNameAgreement(t *testing.T) {
	t.Parallel()

	const name = `"FirePenIncreasedByUncappedFireRes"`

	for _, f := range []struct{ path, role string }{
		{"mod_parser.go", "producer (ModParser.lua:5071)"},
		{"calc_offence.go", "consumer (CalcOffence.lua:725-731)"},
	} {
		src, err := os.ReadFile(f.path)
		if err != nil {
			t.Fatalf("REFUSING: cannot read %s: %v", f.path, err)
		}
		n := 0
		for _, line := range strings.Split(string(src), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			n += strings.Count(line, name)
		}
		if n == 0 {
			t.Errorf("%s: %s has no live occurrence of %s — if this side was renamed, "+
				"the other side is now a dead branch that nothing can fail on", f.path, f.role, name)
		}
	}
}
