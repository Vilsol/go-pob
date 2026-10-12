package calculator

import "testing"

// TestSirusNormalizesToPinnacle pins ConfigTab.lua:804's third gsub.
func TestSirusNormalizesToPinnacle(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"Sirus", "Pinnacle"},
		{"sirus", "Pinnacle"},
		{"Shaper", "Pinnacle"},
		{"Uber Atziri", "Boss"},
		{"Uber", "Uber"},
		{"", "Pinnacle"},
	} {
		if got := normalizeEnemyIsBoss(tc.in); got != tc.want {
			t.Errorf("normalizeEnemyIsBoss(%q) = %q, want %q (ConfigTab.lua:804)", tc.in, got, tc.want)
		}
	}
}
