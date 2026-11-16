package prices

import (
	"strings"
	"testing"
)

// TestSafeNameIsAWindowsFileName: a cache folder named after a host with a port, or a league with punctuation, holds
// no character Windows refuses in a path element.
func TestSafeNameIsAWindowsFileName(t *testing.T) {
	for _, in := range []string{"localhost:8787", `a/b\c.d`, `x*y?z"<w>|v`} {
		if got := safeName(in); strings.ContainsAny(got, `/\.:*?"<>|`) {
			t.Errorf("safeName(%q) = %q", in, got)
		}
	}
}
