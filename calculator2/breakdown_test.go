package calculator2

import (
	"math"
	"testing"
)

// The expectations are the oracle LuaJIT's string.format output for the same arguments.
func TestSFormatMatchesLuaJIT(t *testing.T) {
	got := sFormat("%d %d %d|%g|%g|%.2f|%.2f|%.0f|%g|%s|%g|%.0f|%.3f", 2.7, -2.7, 1e10, 0.1+0.2, 1741.0, 0.125, 1.005, 0.5, 1234567.0, 1.5, 1e6, math.Inf(1), math.Inf(-1))
	want := "2 -2 10000000000|0.3|1741|0.13|1.00|1|1.23457e+06|1.5|1e+06|inf|-inf"
	if got != want {
		t.Errorf("sFormat = %q, want %q", got, want)
	}
}

// LuaJIT's %d on the oracle's x86 truncates with cvttsd2si; Go's int64(NaN) is 0 on arm64 and wasm.
func TestSFormatIntegerOfNonFiniteMatchesLuaJIT(t *testing.T) {
	got := sFormat("%d|%d|%d|%d", math.NaN(), math.Inf(1), math.Inf(-1), 1e19)
	want := "-9223372036854775808|-9223372036854775808|-9223372036854775808|-9223372036854775808"
	if got != want {
		t.Errorf("sFormat = %q, want %q", got, want)
	}
}
