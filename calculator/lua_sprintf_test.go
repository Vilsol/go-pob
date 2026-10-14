package calculator

import "testing"

// TestLuaSprintfRoundsTiesLikeLuaJIT: Expected strings are LuaJIT 2.1's string.format output for the same arguments.
func TestLuaSprintfRoundsTiesLikeLuaJIT(t *testing.T) {
	for _, c := range []struct {
		format string
		args   []any
		want   string
	}{
		{"%.1f", []any{41.25}, "41.3"},
		{"%.1f", []any{0.25}, "0.3"},
		{"%.2f", []any{0.625}, "0.63"},
		{"%.2f per second", []any{34.125}, "34.13 per second"},
		{"%.0f %.0f %.0f", []any{0.5, 1.5, 2.5}, "1 2 3"},
		{"%.1f", []any{-41.25}, "-41.3"},
		{"%.1f", []any{2.45}, "2.5"},
		{"%.1f", []any{41.35}, "41.4"},
		{"%d%% x %.1f%%", []any{3, 0.75}, "3% x 0.8%"},
		{"%s: %.1f", []any{"a", 0.05}, "a: 0.1"},
		{"%5.1f|%-6.2f|", []any{1.25, 0.125}, "  1.3|0.13  |"},
		{"%.2f", []any{0.62}, "0.62"},
		{"%f", []any{0.0000005}, "0.000000"},
		{"%f", []any{0.0000015}, "0.000002"},
	} {
		if got := luaSprintf(c.format, c.args...); got != c.want {
			t.Errorf("luaSprintf(%q, %v) = %q, want %q", c.format, c.args, got, c.want)
		}
	}
}
